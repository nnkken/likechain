package txs

import (
	"encoding/json"
	"fmt"

	"github.com/likecoin/likechain/abci/types"
	"github.com/likecoin/likechain/abci/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// JSONSignature is the signature format using deterministic JSON as message representation
type JSONSignature [65]byte

func (sig *JSONSignature) String() string {
	return common.ToHex(sig[:])
}

func recoverEthSignature(hash []byte, sig [65]byte) (*types.Address, error) {
	// Transform yellow paper V from 27/28 to 0/1
	sig[64] -= 27
	pubKeyBytes, err := crypto.Ecrecover(hash, sig[:])
	if err != nil {
		return nil, err
	}
	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return nil, err
	}
	ethAddr := crypto.PubkeyToAddress(*pubKey)
	addr := types.Address(ethAddr)
	return &addr, nil
}

// JSONMapToHash takes a map[string]interface{} representing a JSON object, returns the hash for signing the message
func JSONMapToHash(jsonMap map[string]interface{}) ([]byte, error) {
	msg, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, err
	}
	fmt.Printf("JSONMap: \"%s\"\n", string(msg))
	sigPrefix := "\x19Ethereum Signed Message:\n"
	hashingMsg := []byte(fmt.Sprintf("%s%d%s", sigPrefix, len(msg), msg))
	return crypto.Keccak256(hashingMsg), nil
}

// RecoverAddress recover the signature to address by the deterministic JSON representation of the message
func (sig *JSONSignature) RecoverAddress(jsonMap map[string]interface{}) (*types.Address, error) {
	hash, err := JSONMapToHash(jsonMap)
	if err != nil {
		return nil, err
	}
	addr, err := recoverEthSignature(hash, *sig)
	return addr, err
}

// Sig transforms a hex string into [65]byte which could be converted into signatures, panic if the string is not a
// valid signature
func Sig(sigHex string) (sig JSONSignature) {
	sigBytes, err := utils.Hex2Bytes(sigHex)
	if err != nil {
		panic(err)
	}
	if len(sigBytes) != 65 {
		panic("Invalid signature length")
	}
	copy(sig[:], sigBytes)
	return sig
}
