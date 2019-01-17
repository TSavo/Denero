// Copyright 2017-2018 DERO Project. All rights reserved.
// Use of this source code in any form is governed by RESEARCH license.
// license can be found in the LICENSE file.
// GPG: 0F39 E425 8C65 3947 702A  8234 08B2 0360 A03A 9DE8
//
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY
// EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL
// THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
// PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
// STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF
// THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package dvm

import "fmt"
import "text/scanner"
import "strings"
import "strconv"
import "unicode"
import "unicode/utf8"
import "go/ast"
import "go/parser"
import "go/token"
import "math"

import "runtime/debug"
import "github.com/deroproject/derosuite/crypto"
import "github.com/deroproject/derosuite/address"

type Vtype int

const (
	Invalid Vtype = iota // default is i invalid
	Uint64               // uint64 data type
	String               // string

	Blob // an encrypted blob, used to add assets/DERO to blockchain without knowing address

	// this should be depreceated
	//ID                   // represents a BLID, TXID, etc
	Address // a DERO address
)

var replacer = strings.NewReplacer("< =", "<=", "> =", ">=", "= =", "==", "! =", "!=", "& &", "&&", "| |", "||", "< <", "<<", "> >", ">>", "< >", "!=")

// Some global variables are always accessible, namely
// SCID  TXID which installed the SC
// TXID  current TXID under which this SC is currently executing
// BLID  current BLID under which TXID is found, THIS CAN be used as deterministic RND Generator, if the SC needs secure randomness
// BL_HEIGHT current height of blockchain

type Variable struct {
	Name  string      `msgpack:"N,omitempty" json:"N,omitempty"`
	Type  Vtype       `msgpack:"T,omitempty" json:"T,omitempty"` // we have only 4 data types
	Value interface{} `msgpack:"V,omitempty" json:"V,omitempty"`
}

type Function struct {
	Name        string              `msgpack:"N,omitempty" json:"N,omitempty"`
	Params      []Variable          `msgpack:"P,omitempty" json:"P,omitempty"`
	ReturnValue Variable            `msgpack:"R,omitempty" json:"R,omitempty"`
	Lines		[]Line				`msgpack:"L,omitempty" json:"L,omitempty"`
	LabelMap	map[string]int	`msgpack:"M,omitempty" json:"M,omitempty"`
}

type Line struct {
	Label 		string 	`msgpack:"L,omitempty" json:"L,omitempty'`
	Code		string  `msgpack:"L,omitempty" json:"L,omitempty'`
}

const LIMIT_interpreted_lines = 2000 // testnet has hardcoded limit
const LIMIT_evals = 11000            // testnet has hardcoded limit eval limit

// each smart code is nothing but a collection of functions
type SmartContract struct {
	Functions map[string]Function `msgpack:"F,omitempty" json:"F,omitempty"`
}

// we have a rudimentary line by line parser
// SC authors must make sure code coverage is 100 %
// we are doing away with AST
func ParseSmartContract(src_code string) (SC SmartContract, pos string, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Recovered in function %+v", r)
		}
	}()

	var s scanner.Scanner
	s.Init(strings.NewReader(src_code))
	s.Filename = "code"
	s.Mode = scanner.ScanIdents | scanner.ScanFloats | scanner.ScanChars | scanner.ScanStrings | scanner.ScanRawStrings | scanner.SkipComments | scanner.ScanComments //  skip comments

	skip_line := int32(-1)
	var current_line int32 = -1
	var line_tokens []string
	var current_function *Function

	SC.Functions = map[string]Function{}

	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		pos := s.Position.String()
		txt := s.TokenText()

		if strings.HasPrefix(txt, ";") || strings.HasPrefix(txt, "REM") { // skip  line, if this is the first word
			skip_line = int32(s.Position.Line)
		}
		if skip_line == int32(s.Position.Line) {
			continue
		}

		/*if strings.HasPrefix(txt, "//") || strings.HasPrefix(txt, "/*") { // skip comments
			continue
		}*/

	process_token:
		if current_line == -1 {
			current_line = int32(s.Position.Line)
		}

		if current_line == int32(s.Position.Line) { // collect a complete line
			line_tokens = append(line_tokens, txt)
		} else { // if new line found, process previous line
			if err = parse_function_line(&SC, &current_function, line_tokens); err != nil {
				return SC, pos, err
			}
			line_tokens = line_tokens[:0]

			current_line = -1
			goto process_token
		}
		//  fmt.Printf("%s: %s line %+v\n", s.Position, txt, line_tokens)
	}

	if len(line_tokens) > 0 { // last line  is processed here
		if err = parse_function_line(&SC, &current_function, line_tokens); err != nil {
			return SC, pos, err
		}
	}

	if current_function != nil {
		err = fmt.Errorf("EOF reached but End Function is missing \"%s\"", current_function.Name)
		return SC, pos, err
	}

	return
}

// checks whether a function name is valid
// a valid name starts with a non digit and does not contain .
func check_valid_name(name string) bool {
	r, size := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError || size == 0 {
		return false
	}
	return unicode.IsLetter(r)
}

func check_valid_type(name string) Vtype {
	switch name {
	case "Uint64":
		return Uint64
	case "Address":
		return Address
	case "Blob":
		return Blob
	case "String":
		return String
	}
	return Invalid
}

// this will parse 1 line at a time, if there is an error, it is returned
func parse_function_line(SC *SmartContract, function **Function, line []string) (err error) {
	pos := 0
	//fmt.Printf("parsing function line %+v\n", line)

	if *function == nil { //if no current function, only legal is Sub
		if !strings.EqualFold(line[pos], "Function") {
			return fmt.Errorf("Expecting declaration of function  but found \"%s\"", line[0])
		}
		pos++

		var f Function

		if len(line) < (pos + 1) {
			return fmt.Errorf("function name missing")
		}

		if !check_valid_name(line[pos]) {
			return fmt.Errorf("function name \"%s\" contains invalid characters", line[pos])
		}
		f.Name = line[pos]

		pos++
		if len(line) < (pos+1) || line[pos] != "(" {
			return fmt.Errorf("function \"%s\" missing '('", f.Name)
		}

	parse_params: // now lets parse function params, but lets filter out ,
		pos++
		if len(line) < (pos + 1) {
			return fmt.Errorf("function \"%s\" missing function parameters", f.Name)
		}

		if line[pos] == "," {
			goto parse_params
		}
		if line[pos] == ")" {
			// function does not have any parameters
			// or all parameters have been parsed

		} else { // we must parse param name, param type as  pairs
			if len(line) < (pos + 2) {
				return fmt.Errorf("function \"%s\" missing function parameters", f.Name)
			}

			param_name := line[pos]
			param_type := check_valid_type(line[pos+1])

			if !check_valid_name(param_name) {
				return fmt.Errorf("function name \"%s\", variable name \"%s\" contains invalid characters", f.Name, param_name)
			}

			if param_type == Invalid {
				return fmt.Errorf("function name \"%s\", variable type \"%s\" is invalid", f.Name, line[pos+1])
			}
			f.Params = append(f.Params, Variable{Name: param_name, Type: param_type})

			pos++
			goto parse_params
		}

		pos++

		// check if we have return value
		if len(line) < (pos + 1) { // we do not have return value
			f.ReturnValue.Type = Invalid
		} else {
			return_type := check_valid_type(line[pos])
			if return_type == Invalid {
				return fmt.Errorf("function name \"%s\", return type \"%s\" is invalid", f.Name, line[pos])
			}
			f.ReturnValue.Type = return_type
		}

		*function = &f
		return nil
	} else if strings.EqualFold(line[pos], "End") && strings.EqualFold(line[pos+1], "Function") {
		SC.Functions[(*function).Name] = **function
		*function = nil
	} else if strings.EqualFold(line[pos], "Function") {
		return fmt.Errorf("Nested functions are not allowed")
	} else {
		possible_line_number := strings.Fields(line[pos])[0]

		_, err := strconv.ParseUint(possible_line_number, 10, 64)
		label := ""
		if err != nil {
			if(strings.Contains(possible_line_number, ":")){
				if((*function).LabelMap[possible_line_number] != 0) { // duplicate line number
					return fmt.Errorf("Error: duplicate label within function  \"%s\" ", (*function).Name)
				}
				label = strings.TrimSuffix(possible_line_number, ":");
			}
		}else{
			label = possible_line_number
		}
		line_copy := line[pos]
		this_line := Line{}
		if(label != ""){
			(*function).LabelMap[label] = pos;
			this_line.Label = label
			this_line.Code = strings.Join(strings.Fields(line_copy)[1:], " ")
		}else{
			this_line.Code = line_copy
		}
		(*function).Lines[pos] = this_line
	}

	return nil
}

// this will run a function from a loaded SC and execute it if possible
// it can run all internal functions
// parameters must be passed as strings
func runSmartContract_internal(SC *SmartContract, EntryPoint string, state *Shared_State, params map[string]interface{}) (result Variable, err error) {
	// if smart contract does not contain function, trigger exception
	function_call, ok := SC.Functions[EntryPoint]
	if !ok {
		err = fmt.Errorf("function \"%s\" is not available in SC", EntryPoint)
		return
	}

	var dvm DVM_Interpreter
	dvm.SC = SC
	dvm.f = function_call
	dvm.Locals = map[string]Variable{}

	dvm.State = state // set state to execute current function

	// parse parameters, rename them, make them available as local variables
	for _, p := range function_call.Params {
		variable := Variable{Name: p.Name, Type: p.Type}
		value, ok := params[p.Name]
		if !ok { // necessary parameter is missing from arguments
			err = fmt.Errorf("Argument \"%s\" is missing while invoking \"%s\"", p.Name, EntryPoint)
			return
		}

		// now lets parse the data,Uint64,Address,String,Blob
		switch p.Type {
		case Uint64:
			variable.Value, err = strconv.ParseUint(value.(string), 0, 64)
			if err != nil {
				return
			}
		case String:
			variable.Value = value.(string)

		case Address, Blob:
			variable.Value = value.(string)
			//panic("address and blob cannot have parameters")

		}

		dvm.Locals[variable.Name] = variable
	}

	// all variables have been collected, start interpreter
	dvm.ReturnValue = dvm.f.ReturnValue // enforce return value to be of same type

	dvm.State.Monitor_recursion++ // higher recursion

	err = dvm.interpret_SmartContract()
	if err != nil {
		return
	}

	result = dvm.ReturnValue

	return
}

// it is similar to internal functions, however it enforces the condition that only Exportable functions are callable
// any function which has first character ASCII and upper case  is considered an exported function
func RunSmartContract(SC *SmartContract, EntryPoint string, state *Shared_State, params map[string]interface{}) (result Variable, err error) {
	// if smart contract does not contain function, trigger exception

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Recovered in function %+v", r)
			//fmt.Printf("%+v ", err)
			fmt.Sprintf("%s\n", debug.Stack())
		}
	}()

	r, size := utf8.DecodeRuneInString(EntryPoint)

	if r == utf8.RuneError || size == 0 {
		return result, fmt.Errorf("Invalid function name")

	}

	if r >= unicode.MaxASCII {
		return result, fmt.Errorf("Invalid function name, First character must be ASCII alphabet")
	}

	if !unicode.IsLetter(r) {
		return result, fmt.Errorf("Invalid function name, First character must be ASCII Letter")
	}

	if !unicode.IsUpper(r) {
		return result, fmt.Errorf("Invalid function name, First character must be Capital/Upper Case")
	}

	// initialize RND
	if state.Monitor_recursion == 0 {
		state.RND = Initialize_RND(state.Chain_inputs.SCID, state.Chain_inputs.BLID, state.Chain_inputs.TXID)
		//state.store = Initialize_TX_store()
		state.DERO_Transfer = map[string]uint64{}
	}

	result, err = runSmartContract_internal(SC, EntryPoint, state, params)

	if err != nil {
		return result, err
	}
	if state.Monitor_recursion != 0 { // recursion must be zero at end
		return result, fmt.Errorf("Invalid  recursion level %d", state.Monitor_recursion)
	}

	// if recursion level is zero, we should check return value and persist the state changes

	return result, err
}

// this structure is all the inputs that are available to SC during execution
type Blockchain_Input struct {
	SCID          crypto.Key      // current smart contract which is executing
	BLID          crypto.Key      // BLID
	TXID          crypto.Key      // current TXID under which TX
	Signer        address.Address // address which signed this
	BL_HEIGHT     uint64          // current chain height under which current tx is valid
	BL_TOPOHEIGHT uint64          // current block topo height which can be used to  uniquely pinpoint the block
}

// all DVMs triggered by the first call, will share this structure
// sharing this structure means RND number attacks are not possible
// all storage state is shared, this means something similar to solidity delegatecall
// this is necessary to prevent number of attacks
type Shared_State struct {
	Persistance bool // whether the results will be persistant or it's just a demo/test call

	Chain_inputs *Blockchain_Input // all blockchain info is available here

	DERO_Balance uint64 // DERO balance of this smart contract, this is loaded
	// this includes any DERO that has arrived with this TX
	DERO_Received uint64 // amount of DERO received with this TX

	DERO_Transfer map[string]uint64 // any DERO that this TX wants to send OUT
	// transfers are only processed after the contract has terminated successfully

	RND   *RND        // this is initialized only once  while invoking entrypoint
	Store *TX_Storage // mechanism to access a data store, can discard changes

	Monitor_recursion         int64 // used to control recursion amount 64 calls are more than necessary
	Monitor_lines_interpreted int64 // number of lines interpreted
	Monitor_ops               int64 // number of ops evaluated, for expressions, variables

}

type DVM_Interpreter struct {
	SCID        string
	SC          *SmartContract
	EntryPoint  string
	function           Function
	IP          uint64              // current line number
	ReturnValue Variable            // Result of current function call
	Locals      map[string]Variable // all local variables

	Chain_inputs *Blockchain_Input // all blockchain info is available here

	State *Shared_State // all shared state between  DVM is available here

	RND *RND // this is initialized only once  while invoking entrypoint

	store *TX_Storage // mechanism to access a data store, can discard changes

}

func (i *DVM_Interpreter) incrementIP(newip uint64) (line []string, err error) {

	i.State.Monitor_lines_interpreted++ // increment line interpreted
	i.IP++
	return
}

// this runs a smart contract function with specific params
func (i *DVM_Interpreter) interpret_SmartContract() (err error) {

	newIP := uint64(0)
	for {
		var line []string
		line, err = i.incrementIP(newIP)
		if err != nil {
			return
		}

		newIP = 0 // this is necessary otherwise, it will trigger an infinite loop in the case given below

		/*
		                 * Function SetOwner(value Uint64, newowner String) Uint64
			10  IF LOAD("owner") == SIGNER() THEN GOTO 30
			20  RETURN 1
			30  STORE("owner",newowner)
			40  RETURN 0
			End Function
		*/

		if i.State.Monitor_lines_interpreted > LIMIT_interpreted_lines {
			panic(fmt.Sprintf("%d lines interpreted, reached limit %d", LIMIT_interpreted_lines, LIMIT_interpreted_lines))
		}

		//fmt.Printf("interpreting line %+v\n", line)

		//fmt.Printf("received line to interpret %+v err\n", line, err)
		switch {
		case strings.EqualFold(line[0], "DIM"):
			newIP, err = i.interpret_DIM(line[1:])
		case strings.EqualFold(line[0], "LET"):
			newIP, err = i.interpret_LET(line[1:])
		case strings.EqualFold(line[0], "GOTO"):
			newIP, err = i.interpret_GOTO(line[1:])
		case strings.EqualFold(line[0], "IF"):
			newIP, err = i.interpret_IF(line[1:])
		case strings.EqualFold(line[0], "RETURN"):
			newIP, err = i.interpret_RETURN(line[1:])

		//ability to print something for debugging purpose
		case strings.EqualFold(line[0], "PRINT"):
			fallthrough
		case strings.EqualFold(line[0], "PRINTF"):
			newIP, err = i.interpret_PRINT(line[1:])

			// if we are here, the first part is unknown

		default:

			// we should try to evaluate expression and make sure it's  a function call
			// now lets evaluate the expression

			expr, err1 := parser.ParseExpr(replacer.Replace(strings.Join(line, " ")))
			if err1 != nil {
				err = err1
				return
			}

			if _, ok := expr.(*ast.CallExpr); !ok {
				return fmt.Errorf("not a function call line %+v\n", line)
			}
			i.eval(expr)

		}

		if err != nil {
			err = fmt.Errorf("err while interpreting line %+v err %s\n", line, err)
			return
		}
		if newIP == math.MaxUint64 {
			break
		}
	}
	return
}

// this is very limited and can be used print only variables
func (dvm *DVM_Interpreter) interpret_PRINT(args []string) (newIP uint64, err error) {
	var variable Variable
	var ok bool
	if len(args) > 0 {
		params := []interface{}{}
		for i := 1; i < len(args); i++ {
			if variable, ok = dvm.Locals[args[i]]; !ok { // TODO what about printing globals
				/*if variable,ok := dvm.Locals[exp.Name];!ok{

				  }*/

			}
			if ok {
				params = append(params, variable.Value)
			} else {
				params = append(params, fmt.Sprintf("unknown variable %s", args[i]))
			}
		}

		_, err = fmt.Printf(strings.Trim(args[0], "\"")+"\n", params...)
	}
	return
}

// process DIM line
func (dvm *DVM_Interpreter) interpret_DIM(line []string) (newIP uint64, err error) {

	if len(line) <= 2 || !strings.EqualFold(line[len(line)-2], "as") {
		return 0, fmt.Errorf("Invalid DIM syntax")
	}

	// check last data type
	data_type := check_valid_type(line[len(line)-1])
	if data_type == Invalid {
		return 0, fmt.Errorf("function name \"%s\", No such Data type \"%s\"", dvm.f.Name, line[len(line)-1])
	}

	for i := 0; i < len(line)-2; i++ {
		if line[i] != "," { // ignore separators

			if !check_valid_name(line[i]) {
				return 0, fmt.Errorf("function name \"%s\", variable name \"%s\" contains invalid characters", dvm.f.Name, line[i])
			}

			// check whether variable is already defined
			if _, ok := dvm.Locals[line[i]]; ok {
				return 0, fmt.Errorf("function name \"%s\", variable name \"%s\" contains invalid characters", dvm.f.Name, line[i])
			}

			// all data variables are pre-initialized

			switch data_type {
			case Uint64:
				dvm.Locals[line[i]] = Variable{Name: line[i], Type: Uint64, Value: uint64(0)}
			case String:
				dvm.Locals[line[i]] = Variable{Name: line[i], Type: String, Value: ""}
			case Address:
				dvm.Locals[line[i]] = Variable{Name: line[i], Type: Address, Value: ""}
			case Blob:
				dvm.Locals[line[i]] = Variable{Name: line[i], Type: Blob, Value: ""}
				// return 0, fmt.Errorf("function name \"%s\", variable name \"%s\" blobs are currently not supported", dvm.f.Name, line[i])

			}
			// fmt.Printf("Initialising variable %s %+v\n",line[i],dvm.Locals[line[i]])

		}
	}

	return
}

// process LET statement
func (dvm *DVM_Interpreter) interpret_LET(line []string) (newIP uint64, err error) {

	if len(line) <= 2 || !strings.EqualFold(line[1], "=") {
		err = fmt.Errorf("Invalid LET syntax")
		return
	}

	if _, ok := dvm.Locals[line[0]]; !ok {
		err = fmt.Errorf("function name \"%s\", variable name \"%s\"  is used without definition", dvm.f.Name, line[0])
		return
	}
	result := dvm.Locals[line[0]]

	expr, err := parser.ParseExpr(strings.Join(line[2:], " "))
	if err != nil {
		return
	}

	expr_result := dvm.eval(expr)
	//fmt.Printf("expression %s = %+v\n", line[0],expr_result)

	//fmt.Printf(" %+v \n", dvm.Locals[line[0]])
	switch result.Type {
	case Uint64:
		result.Value = expr_result.(uint64)
	case String:
		result.Value = expr_result.(string)

	case Blob:
		result.Value = expr_result // FIXME address validation should be provided

	case Address:
		result.Value = expr_result // FIXME address validation should be provided
	}

	dvm.Locals[line[0]] = result
	//  fmt.Printf(" %+v \n", dvm.Locals[line[0]])

	return
}

// process GOTO line
func (dvm *DVM_Interpreter) interpret_GOTO(line []string) (newIP uint64, err error) {

	if len(line) != 1 {
		err = fmt.Errorf("GOTO  contains 1 mandatory line number as argument")
		return
	}

	newIP, err = strconv.ParseUint(line[0], 0, 64)
	if err != nil {
		return
	}

	if newIP == 0 || newIP == math.MaxUint64 {
		return 0, fmt.Errorf("GOTO  has invalid line number \"%d\"", newIP)
	}
	return
}

// process IF line
// IF has two forms  vis  x,y are line numbers
// IF expr THEN GOTO x
// IF expr THEN GOTO x ELSE GOTO y
func (dvm *DVM_Interpreter) interpret_IF(line []string) (newIP uint64, err error) {

	thenip := uint64(0)
	elseip := uint64(0)

	// first form of IF
	if len(line) >= 4 && strings.EqualFold(line[len(line)-3], "THEN") && strings.EqualFold(line[len(line)-2], "GOTO") {

		thenip, err = strconv.ParseUint(line[len(line)-1], 0, 64)
		if err != nil {
			return
		}
		line = line[:len(line)-3]

	} else if len(line) >= 7 && strings.EqualFold(line[len(line)-6], "THEN") && strings.EqualFold(line[len(line)-5], "GOTO") && strings.EqualFold(line[len(line)-3], "ELSE") && strings.EqualFold(line[len(line)-2], "GOTO") {

		thenip, err = strconv.ParseUint(line[len(line)-4], 0, 64)
		if err != nil {
			return
		}

		elseip, err = strconv.ParseUint(line[len(line)-1], 0, 64)
		if err != nil {
			return
		}

		if elseip == 0 || elseip == math.MaxUint64 {
			return 0, fmt.Errorf("ELSE GOTO  has invalid line number \"%d\"", thenip)
		}

		line = line[:len(line)-6]
	} else {
		err = fmt.Errorf("Invalid IF syntax")
		return
	}

	if thenip == 0 || thenip == math.MaxUint64 {
		return 0, fmt.Errorf("THEN GOTO  has invalid line number \"%d\"", thenip)
	}

	// now lets evaluate the expression

	expr, err := parser.ParseExpr(replacer.Replace(strings.Join(line, " ")))
	if err != nil {
		return
	}

	expr_result := dvm.eval(expr)
	//fmt.Printf("if %d %T expr( %s)\n", expr_result, expr_result, replacer.Replace(strings.Join(line, " ")))
	if result, ok := expr_result.(uint64); ok {
		if result != 0 {
			newIP = thenip
		} else {
			newIP = elseip
		}
	} else {

		err = fmt.Errorf("Invalid IF expression  \"%s\"", replacer.Replace(strings.Join(line, " ")))
	}

	return

}

// process RETURN line
func (dvm *DVM_Interpreter) interpret_RETURN(line []string) (newIP uint64, err error) {

	if dvm.ReturnValue.Type == Invalid {
		if len(line) != 0 {
			err = fmt.Errorf("function name \"%s\" cannot return anything", dvm.f.Name)
			return
		}

		dvm.State.Monitor_recursion-- // lower recursion
		newIP = math.MaxUint64        // simple return
		return
	}

	if len(line) == 0 {
		err = fmt.Errorf("function name \"%s\" should return  a value", dvm.f.Name)
		return
	}

	// we may be returning an expression which must be solved
	expr, err := parser.ParseExpr(replacer.Replace(strings.Join(line, " ")))
	if err != nil {
		return
	}

	expr_result := dvm.eval(expr)
	//fmt.Printf("expression %+v %T\n", expr_result, expr_result)

	switch dvm.ReturnValue.Type {
	case Uint64:
		dvm.ReturnValue.Value = expr_result.(uint64)
	case String:
		dvm.ReturnValue.Value = expr_result.(string)
	case Blob:
		dvm.ReturnValue.Value = expr_result.(string)
	case Address:
		dvm.ReturnValue.Value = expr_result.(string)

	}

	dvm.State.Monitor_recursion-- // lower recursion
	newIP = math.MaxUint64        // simple return

	return
}

// only returns identifiers
func (dvm *DVM_Interpreter) eval_identifier(exp ast.Expr) string {

	switch exp := exp.(type) {
	case *ast.Ident: // it's a variable,
		return exp.Name
	default:
		panic("expecting identifier")

	}

}

func (dvm *DVM_Interpreter) eval(exp ast.Expr) interface{} {

	dvm.State.Monitor_ops++ // maintain counter

	if dvm.State.Monitor_ops > LIMIT_evals {
		panic(fmt.Sprintf("%d lines interpreted, evals reached limit %d", dvm.State.Monitor_lines_interpreted, LIMIT_evals))
	}

	//fmt.Printf("exp %+v  %T\n", exp, exp)
	switch exp := exp.(type) {
	case *ast.ParenExpr:
		return dvm.eval(exp.X)

	case *ast.UnaryExpr: // there are 2 unary operators, one is binary NOT , second is logical not
		switch exp.Op {
		case token.XOR:
			return ^(dvm.eval(exp.X).(uint64))
		case token.NOT:
			x := dvm.eval(exp.X)
			switch x := x.(type) {
			case uint64:
				return ^x
			case string:
				if IsZero(x) == 1 {
					return uint64(1)
				}
				return uint64(0)

			}
		}

	case *ast.BinaryExpr:
		return dvm.evalBinaryExpr(exp)
	case *ast.Ident: // it's a variable,
		if _, ok := dvm.Locals[exp.Name]; !ok {
			panic(fmt.Sprintf("function name \"%s\", variable name \"%s\"  is used without definition", dvm.f.Name, exp.Name))

		}
		//fmt.Printf("value %s %d\n",exp.Name,  dvm.Locals[exp.Name].Value)
		return dvm.Locals[exp.Name].Value

	// there are 2 types of calls, one within the smartcontract
	// other one crosses smart contract boundaries
	case *ast.CallExpr:
		func_name := dvm.eval_identifier(exp.Fun)
		//fmt.Printf("Call expression %+v %s \"%s\" \n",exp,exp.Fun, func_name)
		// if call is internal
		//

		// try to handle internal functions, SC function cannot overide internal functions
		if ok, result := dvm.Handle_Internal_Function(exp, func_name); ok {
			return result
		}
		function_call, ok := dvm.SC.Functions[func_name]
		if !ok {
			panic(fmt.Sprintf("Unknown function called \"%s\"", exp.Fun))
		}
		if len(function_call.Params) != len(exp.Args) {
			panic(fmt.Sprintf("function \"%s\" called with incorrect number of arguments , expected %d , actual %d", func_name, len(function_call.Params), len(exp.Args)))
		}

		arguments := map[string]interface{}{}
		for i, p := range function_call.Params {
			switch p.Type {
			case Uint64:
				arguments[p.Name] = fmt.Sprintf("%d", dvm.eval(exp.Args[i]).(uint64))
			case String:
				arguments[p.Name] = dvm.eval(exp.Args[i]).(string)

			case Blob, Address:
				arguments[p.Name] = dvm.eval(exp.Args[i]).(string)
			}
		}

		// allow calling unexported functions
		result, err := runSmartContract_internal(dvm.SC, func_name, dvm.State, arguments)
		if err != nil {
			panic(err)
		}
		if function_call.ReturnValue.Type != Invalid {
			return result.Value
		}
		return nil

	case *ast.BasicLit:
		switch exp.Kind {
		case token.INT:
			i, err := strconv.ParseUint(exp.Value, 0, 64)
			if err != nil {
				panic(err)
			}
			return i
		case token.STRING:
			unquoted, err := strconv.Unquote(exp.Value)
			if err != nil {
				panic(err)
			}
			return unquoted
		}
	default:
		panic(fmt.Sprintf("Unhandled expression type %+v", exp))

	}

	panic("We should never reach here while evaluating expressions")
	return 0
}

// this can be used to check whether variable has a default value
// for uint64 , it is 0
// for string , it is ""
// TODO Address, Blob
func IsZero(value interface{}) uint64 {
	switch v := value.(type) {
	case uint64:
		if v == 0 {
			return 1
		}
	case string:
		if v == "" {
			return 1
		}

	default:
		panic("IsZero not being handled")

	}

	return 0
}

func (dvm *DVM_Interpreter) evalBinaryExpr(exp *ast.BinaryExpr) interface{} {

	left := dvm.eval(exp.X)
	right := dvm.eval(exp.Y)

	//fmt.Printf("left %d %+v  right %d\n", left, exp.Op, right)

	// special case to append uint64 to strings
	if fmt.Sprintf("%T", left) == "string" && fmt.Sprintf("%T", right) == "uint64" {
		return left.(string) + fmt.Sprintf("%d", right)
	}

	if fmt.Sprintf("%T", left) != fmt.Sprintf("%T", right) {
		panic(fmt.Sprintf("Expressions cannot be different type(String/Uint64) left (val %+v %+v)   right (%+v %+v)", left, exp.X, right, exp.Y))
	}

	// logical ops are handled differently
	switch exp.Op {
	case token.LAND:
		if (IsZero(left) == 0) && (IsZero(right) == 0) { // both sides should be set
			return uint64(1)
		}

		return uint64(0)
	case token.LOR:
		//fmt.Printf("left %d   right %d\n", left,right)
		//fmt.Printf("left %v   right %v\n", (IsZero(left) != 0),(IsZero(right) != 0))
		if (IsZero(left) == 0) || (IsZero(right) == 0) {
			return uint64(1)
		}
		return uint64(0)

	}

	// handle string operands
	if fmt.Sprintf("%T", left) == "string" {
		left_string := left.(string)
		right_string := right.(string)

		switch exp.Op {
		case token.ADD:
			return left_string + right_string
		case token.EQL:
			if left_string == right_string {
				return uint64(1)
			}
			return uint64(0)
		case token.NEQ:
			if left_string != right_string {
				return uint64(1)
			}
			return uint64(0)
		default:
			panic("String data type only support addition operation")
		}

	}

	left_uint64 := left.(uint64)
	right_uint64 := right.(uint64)

	switch exp.Op {
	case token.ADD:
		return left_uint64 + right_uint64 // TODO : can we add rounding case here and raise exception
	case token.SUB:
		return left_uint64 - right_uint64 // TODO : can we add rounding case here and raise exception
	case token.MUL:
		return left_uint64 * right_uint64
	case token.QUO:
		return left_uint64 / right_uint64
	case token.REM:
		return left_uint64 % right_uint64

		//bitwise ops
	case token.AND:
		return left_uint64 & right_uint64
	case token.OR:
		return left_uint64 | right_uint64
	case token.XOR:
		return left_uint64 ^ right_uint64
	case token.SHL:
		return left_uint64 << right_uint64
	case token.SHR:
		return left_uint64 >> right_uint64

	case token.EQL:
		if left_uint64 == right_uint64 {
			return uint64(1)
		}
	case token.NEQ:
		if left_uint64 != right_uint64 {
			return uint64(1)
		}
	case token.LEQ:
		if left_uint64 <= right_uint64 {
			return uint64(1)
		}
	case token.GEQ:
		if left_uint64 >= right_uint64 {
			return uint64(1)
		}
	case token.LSS:
		if left_uint64 < right_uint64 {
			return uint64(1)
		}
	case token.GTR:
		if left_uint64 > right_uint64 {
			return uint64(1)
		}
	default:
		panic("This operation  cannot be handled")
	}
	return uint64(0)
}

/*
func main() {

	const src = `
        Function HelloWorld(s Uint64) Uint64

        5
        10 Dim x1, x2 as Uint64
        20 LET x1 = 3
        25 LET x1 = 3 + 5 - 1
        27 LET x2 = x1 + 3
        28 RETURN HelloWorld2(s*s)
        30 Printf "x1=%d x2=%d s = %d" x1 x2 s
        35 IF x1 == 7 THEN GOTO 100 ELSE GOTO 38
        38 Dim y1, y2 as String
        40 LET y1 = "first string" + "second string"


        60 GOTO 100
        80 GOTO 10
        100 RETURN 0
        500 LET y = 45
        501 y = 45

        End Function


        Function HelloWorld2(s Uint64) Uint64

	900 Return s
	950 y = 45
    // Comment begins at column 5.

   ;  This line should not be included in the output.
REM jj



7000 let x.ku[1+1]=0x55
End Function

`

	// we should be build an AST here

	sc, pos, err := ParseSmartContract(src)
	if err != nil {
		fmt.Printf("Error while parsing smart contract pos %s err : %s\n", pos, err)
		return
	}

	result, err := RunSmartContract(&sc, "HelloWorld", map[string]interface{}{"s": "9999"})

	fmt.Printf("result %+v err %s\n", result, err)

}
*/
