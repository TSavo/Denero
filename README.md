Use and distribution of this technology is subject to the Java Research License included herein.

This is a FORK of the Dero project to add features that Dero users are requesting. The author offers the code to the Dero developers only under the licensing permitted by it's original license: for research purposes only.

Ideally these changes will be considered for inclusion in the 'main Dero  code base', but until such time as that is possible per it's restrictive license and doesn't require actions by it's author explicitly not permitted the license, this repo is maintained as a downstream FORK.

As such, much of the code will be refencing "Dero", not "Denero".

# DENERO: Secure, Private CryptoNote DAG Blockchain with Smart Contracts
## ABOUT DENERO PROJECT
DENERO is decentralized DAG(Directed Acyclic Graph) based blockchain with enhanced reliability, privacy, security, and usability. Consensus algorithm is PoW based on original cryptonight. DERO is industry leading and the first blockchain to have bulletproofs, TLS encrypted Network.

### DENERO blockchain has the following salient features:
 - DAG Based: No orphan blocks, No soft-forks.
 - 12 Second Block time.
 - Extremely fast transactions with 2 minutes confirmation time.
 - SSL/TLS P2P Network.
 - CryptoNote: Fully Encrypted Blockchain
 - BulletProofs: Zero Knowledge range-proofs(NIZK).
 - Ring signatures.
 - Fully Auditable Supply.
 - DENERO blockchain is written from scratch in Golang. 
 - Developed and maintained by original developers.

### DENERO DAG
DENERO DAG implementation builds outs a main chain from the DAG network of blocks which refers to main blocks (100% reward) and side blocks (67% rewards). Side blocks contribute to chain PoW security and thus traditional 51% attacks are not possible on DERO network. If DERO network finds another block at the same height, instead of choosing one, DERO include both blocks. Thus, rendering the 51% attack futile.


Traditional Blockchains process blocks as single unit of computation(if a double-spend tx occurs within the block, entire block is rejected). However DERO network accepts such blocks since DERO blockchain considers transaction as a single unit of computation.DERO blocks may contain duplicate or double-spend transactions which are filtered by client protocol and ignored by the network. DERO DAG processes transactions atomically one transaction at a time.


**DENERO DAG in action**
![DENERO DAG](http://seeds.dero.io/images/Dag1.jpeg)


## Downloads
| Operating System | Download                                 |
| ---------------- | ---------------------------------------- |
| Windows 32       | https://github.com/deroproject/derosuite/releases |
| Windows 64       | https://github.com/deroproject/derosuite/releases |
| Mac 10.8 & Later | https://github.com/deroproject/derosuite/releases |
| Linux 32         | https://github.com/deroproject/derosuite/releases |
| Linux 64         | https://github.com/deroproject/derosuite/releases |
| OpenBSD 64       | https://github.com/deroproject/derosuite/releases |
| FreeBSD 64       | https://github.com/deroproject/derosuite/releases |
| Linux ARM 64     | https://github.com/deroproject/derosuite/releases |
| More Builds      | https://github.com/deroproject/derosuite/releases |


### Build from sources:
In go workspace: **go get -u github.com/tsavo/denero**

Check bin folder for DENEROd, explorer and wallet  binaries. Use golang-1.10.3 version minimum.

### DENERO Quickstart
1. Choose your Operating System and [download DENERO software](https://github.com/tsavo/denero/releases)
2. Extract the file and change to extracted folder in cmd prompt.
3. Start DENEROd daemon and wait to fully sync till prompt goes green.
4. Open new cmd prompt and run DENERO-wallet-cli.


For detailed walk through to create/restore Dero wallet pls see: [Create/Restore DENERO Wallet in one minute](https://forum.dero.io/t/create-backup-restore-dero-wallet-in-one-minute/110)


**DENERO Daemon in action**
![DENERO Daemon](http://seeds.dero.io/images/derod1.png)


**DENERO Wallet in action**
![DENERO Wallet](http://seeds.dero.io/images/wallet1.jpeg)

## Technical
For specific details of current DERO core (daemon) implementation and capabilities, see below:

1. **DAG:** No orphan blocks, No soft-forks.
2. **BulletProofs:** Zero Knowledge range-proofs(NIZK)
3. **Cryptonight Hash:** This is memory-bound algorithm. This provides assurance that all miners are equal. ( No miner has any advantage over common miners).
4. **P2P Protocol:** This layers controls exchange of blocks, transactions and blockchain itself.
5.  **Pederson Commitment:** (Part of ring confidential transactions): Pederson commitment algorithm is a cryptographic primitive that allows user to commit to a chosen value  while keeping it hidden to others. Pederson commitment  is used to hide all amounts without revealing the actual amount. It is a homomorphic commitment scheme.
6.  **Borromean Signature:**  (Part of ring confidential transactions):  Borromean Signatures are used to prove that the commitment has a specific value, without revealing the value itself.
7.  **Additive Homomorphic Encryption:** Additive Homomorphic Encryption is used to prove that sum of encrypted Input transaction amounts is EQUAL to sum of encrypted output amounts. This is based on Homomorphic Pederson commitment scheme.
8.  **Multilayered Linkable Spontaneous Anonymous Group (MLSAG) :** (Part of ring confidential transactions): MLSAG gives DERO untraceability and increases privacy and fungibility. MLSAG is a user controlled parameter ( Mixin) which the user can change to improve his privacy. Mixin of minimal amount is enforced and user cannot disable it.
9.  **Ring Confidential Transactions:** Gives untraceability , privacy and fungibility while making sure that the system is stable and secure.
10.  **Core-Consensus Protocol implemented:** Consensus protocol serves 2 major purpose
   1. Protects the system from adversaries and protects it from forking and tampering.
   2. Next block in the chain is the one and only correct version of truth ( balances).
11.  **Proof-of-Work(PoW) algorithm:**  PoW part of core consensus protocol which is used to cryptographically prove that X amount of work has been done to successfully find a block.
12.  **Difficulty algorithm**: Difficulty algorithm controls the system so as blocks are found roughly at the same speed, irrespective of the number and amount of mining power deployed.
13.  **Serialization/De-serialization of blocks**: Capability to encode/decode/process blocks .
14.  **Serialization/De-serialization of transactions**: Capability to encode/decode/process transactions.
15.  **Transaction validity and verification**: Any transactions flowing within the DERO network are validated,verified.
16.  **Socks proxy:** Socks proxy has been implemented and integrated within the daemon to decrease user identifiability and  improve user anonymity.
17.  **Interactive daemon** can print blocks, txs, even entire blockchain from within the daemon 
18.  **status, diff, print_bc, print_block, print_tx** and several other commands implemented
19.  GO DENERO Daemon has both mainnet, testnet support.
20.  **Enhanced Reliability, Privacy, Security, Useability, Portabilty assured.** For discussion on each point how pls visit forum.

## Crypto
Secure and fast crypto is the basic necessity of this project and adequate amount of time has been devoted to develop/study/implement/audit it. Most of the crypto such as ring signatures have been studied by various researchers and are in production by number of projects. As far as the Bulletproofs are considered, since DERO is the first one to implement/deploy, they have been given a more detailed look. First, a bare bones bulletproofs was implemented, then implementations in development were studied (Benedict Bunz,XMR, Dalek Bulletproofs) and thus improving our own implementation.Some new improvements were discovered and implemented (There are number of other improvements which are not explained here). Major improvements are in the Double-Base Double-Scalar Multiplication while validating bulletproofs. A typical bulletproof takes ~15-17 ms to verify. Optimised bulletproofs takes ~1 to ~2 ms(simple bulletproof, no aggregate/batching). Since, in the case of bulletproofs the bases are fixed, we can use precompute table to convert 64*2 Base Scalar multiplication into doublings and additions (NOTE: We do not use Bos-Coster/Pippienger methods). This time can be again easily decreased to .5 ms with some more optimizations.With batching and aggregation, 5000 range-proofs (~2500 TX) can be easily verified on even a laptop. The implementation for bulletproofs is in github.com/deroproject/derosuite/crypto/ringct/bulletproof.go , optimized version is in github.com/deroproject/derosuite/crypto/ringct/bulletproof_ultrafast.go

There are other optimizations such as base-scalar multiplication could be done in less than a microsecond. Some of these optimizations are not yet deployed and may be deployed at a later stage.

###  About DENERO Rocket Bulletproofs
 - DENERO ultrafast bulletproofs optimization techniques in the form used did not exist anywhere in publicly available cryptography literature at the time of implementation. Please contact for any source/reference to include here if it exists.  Ultrafast optimizations verifies Dero bulletproofs 10 times faster than other/original bulletproof implementations. See: https://github.com/deroproject/derosuite/blob/master/crypto/ringct/bulletproof_ultrafast.go

 - DENERO rocket bulletproof implementations are hardened, which protects DERO from certain class of attacks.  

 - DENERO rocket bulletproof transactions structures are not compatible with other implementations.

Also there are several optimizations planned in near future in Dero rocket bulletproofs which will lead to several times performance boost. Presently they are under study for bugs, verifications, compatibilty etc.

For technical issues and discussion, please visit https://forum.dero.io



