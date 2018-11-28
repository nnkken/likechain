package withdraw

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/tendermint/tendermint/crypto"
	tmRPC "github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"

	"github.com/likecoin/likechain/services/abi/relay"
	"github.com/likecoin/likechain/services/tendermint"
)

// type AppHashContractProof struct {
// 	Height     uint64
// 	Round      uint64
// 	Payload struct {
// 		SuffixLen  uint8
// 		Suffix     []byte
// 		VotesCount uint8
// 		Votes      []struct {
// 			TimeLen uint8
// 			Time    []byte
// 			Sig     [65]byte
// 		}
// 		AppHashLen   uint8
// 		AppHash      []byte
// 		AppHashProof [4][32]byte
// 	}
// }

func extractAppendix(vote *types.CanonicalVote) (time, suffix []byte) {
	cdc := types.GetCodec()
	buf := new(bytes.Buffer)
	buf.WriteByte(0x22)
	buf.Write(cdc.MustMarshalBinaryLengthPrefixed(vote.Timestamp))
	time = buf.Bytes()

	buf = new(bytes.Buffer)
	buf.WriteByte(0x2A)
	buf.Write(cdc.MustMarshalBinaryLengthPrefixed(vote.BlockID))
	buf.WriteByte(0x32)
	buf.Write(cdc.MustMarshalBinaryBare(vote.ChainID))
	suffix = buf.Bytes()
	return time, suffix
}

func genContractProofPayload(signedHeader *types.SignedHeader, tmToEthAddr map[int]common.Address) []byte {
	header := signedHeader.Header
	rawVotes := signedHeader.Commit.Precommits
	votes := []*types.Vote{}

	for _, vote := range rawVotes {
		if vote != nil {
			votes = append(votes, vote)
		}
	}

	votesCount := len(votes)
	if votesCount == 0 {
		return nil
	}

	cv := types.CanonicalizeVote(header.ChainID, votes[0])
	_, suffix := extractAppendix(&cv)

	buf := new(bytes.Buffer)
	buf.WriteByte(uint8(len(suffix)))
	buf.Write(suffix)
	buf.WriteByte(uint8(votesCount))

	for _, vote := range votes {
		cv := types.CanonicalizeVote(header.ChainID, vote)
		time, _ := extractAppendix(&cv)
		buf.WriteByte(uint8(len(time)))
		buf.Write(time)

		signBytes := vote.SignBytes(header.ChainID)
		ethAddr := tmToEthAddr[vote.ValidatorIndex]
		ethSig := tendermint.SignatureToEthereumSig(vote.Signature, crypto.Sha256(signBytes), ethAddr)
		buf.Write(ethSig[64:])
		buf.Write(ethSig[:64])
	}

	buf.WriteByte(uint8(len(header.AppHash)))
	buf.Write(header.AppHash)
	_, proof := Proof(header)
	for _, pf := range proof {
		buf.Write(pf)
	}
	return buf.Bytes()
}

func waitForReceipt(ethClient *ethclient.Client, txHash common.Hash) (*ethTypes.Receipt, error) {
	for {
		receipt, err := ethClient.TransactionReceipt(context.Background(), txHash)
		if receipt != nil {
			return receipt, nil
		}
		if err != nil {
		}
		if err != ethereum.NotFound {
			return nil, err
		}
		time.Sleep(15 * time.Second)
	}
}

func doWithdraw(tmClient *tmRPC.HTTP, ethClient *ethclient.Client, auth *bind.TransactOpts, contractAddr common.Address, callData withdrawCallData) {
	contract, err := relay.NewRelay(contractAddr, ethClient)
	if err != nil {
		panic(err)
	}

	// fmt.Printf("Calling withdraw, withdrawInfo: 0x%v, contractProof: 0x%v\n", cmn.HexBytes(callData.WithdrawInfo), cmn.HexBytes(callData.ContractProof))
	tx, err := contract.Withdraw(auth, callData.WithdrawInfo, callData.ContractProof)
	if err != nil {
		panic(err)
	}

	receipt, err := waitForReceipt(ethClient, tx.Hash())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Withdraw done, status: %v, gas: %v\n", receipt.Status, receipt.GasUsed)
}

func getContractHeight(ethClient *ethclient.Client, contractAddr common.Address) int64 {
	contract, err := relay.NewRelay(contractAddr, ethClient)
	if err != nil {
		panic(err)
	}
	height, err := contract.LatestBlockHeight(nil)
	if err != nil {
		panic(err)
	}
	return height.Int64()
}

func commitWithdrawHash(tmClient *tmRPC.HTTP, ethClient *ethclient.Client, auth *bind.TransactOpts, contractAddr common.Address, height int64) {
	validators := tendermint.GetValidators(tmClient)
	tmToEthAddr := tendermint.MapValidatorIndexToEthAddr(validators)

	signedHeader := tendermint.GetSignedHeader(tmClient, height)

	// fmt.Printf("SignedHeader block hash: %v\n", signedHeader.Commit.BlockID.Hash)
	contractPayload := genContractProofPayload(&signedHeader, tmToEthAddr)
	// fmt.Printf("Calling commitWithdrawHash, contract payload: 0x%v\n", cmn.HexBytes(contractPayload))
	contract, err := relay.NewRelay(contractAddr, ethClient)
	if err != nil {
		panic(err)
	}

	round := uint64(signedHeader.Commit.Round())
	tx, err := contract.CommitWithdrawHash(auth, uint64(height), round, contractPayload)
	if err != nil {
		panic(err)
	}

	receipt, err := waitForReceipt(ethClient, tx.Hash())
	if err != nil {
		panic(err)
	}
	fmt.Printf("CommitWithdrawHash done status: %v, gas: %v\n", receipt.Status, receipt.GasUsed)
}

type withdrawCallData struct {
	WithdrawInfo  []byte
	ContractProof []byte
}

func getWithdrawCallDataArr(tmClient *tmRPC.HTTP, lastHeight, newHeight int64) []withdrawCallData {
	fmt.Printf("Search withdraws with %d < height <= %d\n", lastHeight, newHeight)
	queryString := fmt.Sprintf("withdraw.height>%d AND withdraw.height<=%d", lastHeight, newHeight)
	// TODO: may need pagination
	searchResult, err := tmClient.TxSearch(queryString, true, 1, 100)
	if err != nil {
		panic(err)
	}
	if searchResult.TotalCount <= 0 {
		fmt.Println("No search result")
		return nil
	}
	callDataArr := make([]withdrawCallData, searchResult.TotalCount)
	for i := 0; i < searchResult.TotalCount; i++ {
		packedTx := searchResult.Txs[i].TxResult.Data
		// fmt.Printf("Result %d: %v\n", i, cmn.HexBytes(packedTx))
		queryResult, err := tmClient.ABCIQueryWithOptions("withdraw_proof", packedTx, tmRPC.ABCIQueryOptions{Height: newHeight})
		if err != nil {
			panic(err)
		}
		proof := ParseRangeProof(queryResult.Response.Value)
		if proof == nil {
			panic(fmt.Sprintf("Cannot parse RangeProof: %s", string(queryResult.Response.Value)))
		}
		// fmt.Printf("Proof rootHash: %v\n", cmn.HexBytes(proof.ComputeRootHash()))
		contractProof := proof.ContractProof()
		callDataArr[i] = withdrawCallData{packedTx, contractProof}
	}
	return callDataArr
}

// Run starts the subscription to the withdraws on LikeChain and commits proofs onto Ethereum
func Run(tmClient *tmRPC.HTTP, ethClient *ethclient.Client, auth *bind.TransactOpts, contractAddr common.Address) {
	lastHeight := getContractHeight(ethClient, contractAddr)
	for ; ; time.Sleep(time.Minute) {
		// TODO: load lastHeight from database?
		newHeight := tendermint.GetHeight(tmClient)
		if newHeight == lastHeight {
			fmt.Printf("No new blocks since last height (%d)\n", lastHeight)
			continue
		}
		withdrawCallDataArr := getWithdrawCallDataArr(tmClient, lastHeight, newHeight)
		if len(withdrawCallDataArr) <= 0 {
			continue
		}
		contractHeight := getContractHeight(ethClient, contractAddr)
		if contractHeight < newHeight {
			commitWithdrawHash(tmClient, ethClient, auth, contractAddr, newHeight)
		} else if contractHeight > newHeight {
			panic("contractHeight > newHeight")
		}
		// TODO: save callDataArr in database
		// TODO: save lastHeight in database?
		lastHeight = newHeight
		for _, callData := range withdrawCallDataArr {
			doWithdraw(tmClient, ethClient, auth, contractAddr, callData)
		}
	}
}
