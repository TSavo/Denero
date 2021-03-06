/* Lotter Smart Contract in DVM-BASIC  
   This lottery smart contract  will give lottery wins every  xth try.
*/



        Function Lottery(value Uint64) Uint64
	10  dim deposit_count,winner as Uint64
	20  LET deposit_count =  LOAD("deposit_count")+1
	25  IF value == 0 THEN GOTO 110  // if deposit amount is 0, simply return
	30  STORE("depositor_address" + (deposit_count-1), SIGNER()) // store address for later on payment
	40  STORE("deposit_total", LOAD("deposit_total") + value )
	50  STORE("deposit_count",deposit_count)
	60  IF LOAD("lotteryeveryXdeposit") > deposit_count THEN GOTO 110 // we will wait till X players join in
	  // we are here means all players have joined in, roll the DICE, 
	70  LET winner  = RANDOM() % deposit_count // we have a winner
	80  SEND_DERO_TO_ADDRESS(LOAD("depositor_address" + winner) , LOAD("lotterygiveback")*LOAD("deposit_total")/10000)
	
            // re initialize for another round
        90   STORE("deposit_count", 0)   //  initial players
	100  STORE("deposit_total", 0)   //  total deposit of all players
	110  RETURN 0
	End Function

	
	// this function is used to initialize parameters during install time
	Function Initialize() Uint64
	10  STORE("owner", SIGNER())   // store in DB  ["owner"] = address
	20  STORE("lotteryeveryXdeposit", 2)   // lottery will reward every X deposits
        // how much will lottery giveback in 1/10000 parts, granularity .01 %
	30  STORE("lotterygiveback", 9900)   // lottery will give reward 99% of deposits, 1 % is accumulated for owner to withdraw
	33  STORE("deposit_count", 0)   //  initial players
	34  STORE("deposit_total", 0)   //  total deposit of all players
	35 printf "Initialize executed"
	40 RETURN 0 
	End Function 
	
	
	
        // used to tune lottery parameters
	Function TuneLotteryParameters(input Uint64, lotteryeveryXdeposit Uint64, lotterygiveback Uint64) Uint64
	10  dim key,stored_owner as String
	20  dim value_uint64 as Uint64
	30  IF ADDRESS_RAW(LOAD("owner")) == ADDRESS_RAW(SIGNER()) THEN GOTO 100  // check whether owner is real owner
	40  RETURN 1
	
	100  STORE("lotteryeveryXdeposit", lotteryeveryXdeposit)   // lottery will reward every X deposits
	130  STORE("lotterygiveback", value_uint64)   // how much will lottery giveback in 1/10000 parts, granularity .01 %
	140  RETURN 0 // return success
	End Function
	

	
	// this function is used to change owner 
	// owner is an string form of address 
	Function TransferOwnership(newowner String) Uint64 
	10  IF ADDRESS_RAW(LOAD("owner")) == ADDRESS_RAW(SIGNER()) THEN GOTO 30 
	20  RETURN 1
	30  STORE("tmpowner",newowner)
	40  RETURN 0
	End Function
	
	// until the new owner claims ownership, existing owner remains owner
        Function ClaimOwnership() Uint64 
	10  IF ADDRESS_RAW(LOAD("tmpowner")) == ADDRESS_RAW(SIGNER()) THEN GOTO 30 
	20  RETURN 1
	30  STORE("owner",SIGNER()) // ownership claim successful
	40  RETURN 0
	End Function
	
	// if signer is owner, withdraw any requested funds
	// if everthing is okay, thety will be showing in signers wallet
        Function Withdraw( amount Uint64) Uint64 
	10  IF ADDRESS_RAW(LOAD("owner")) == ADDRESS_RAW(SIGNER()) THEN GOTO 30 
	20  RETURN 1
	30  SEND_DERO_TO_ADDRESS(SIGNER(),amount)
	40  RETURN 0
	End Function
	
	
	


