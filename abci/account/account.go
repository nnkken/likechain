package account

import (
	"bytes"
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/likecoin/likechain/abci/context"
	logger "github.com/likecoin/likechain/abci/log"
	"github.com/likecoin/likechain/abci/types"
	"github.com/likecoin/likechain/abci/utils"
)

var log = logger.L

func getIDAddrPairPrefixKey(id *types.LikeChainID) []byte {
	return utils.PrefixKey([][]byte{
		[]byte("acc"),
		id.Bytes(),
		[]byte("addr"),
	})
}

func getIDAddrPairKey(id *types.LikeChainID, addr *types.Address) []byte {
	return utils.JoinKeys([][]byte{
		[]byte("acc"),
		id.Bytes(),
		[]byte("addr"),
		addr.Bytes(),
	})
}

func getAddrIDPairKey(addr *types.Address) []byte {
	return utils.DbRawKey(addr.Bytes(), "acc", "id")
}

// NewAccount creates a new account
func NewAccount(state context.IMutableState, addr *types.Address) *types.LikeChainID {
	id := generateLikeChainID(state)
	NewAccountFromID(state, id, addr)
	return id
}

// NewAccountFromID creates a new account from a given LikeChain ID
func NewAccountFromID(state context.IMutableState, id *types.LikeChainID, addr *types.Address) {
	// Save address mapping
	state.MutableStateTree().Set(getAddrIDPairKey(addr), id.Bytes())
	state.MutableStateTree().Set(getIDAddrPairKey(id, addr), []byte{})

	// Check if address already has balance
	addrBalance := FetchRawBalance(state, addr)

	var balance *big.Int
	if addrBalance.Cmp(big.NewInt(0)) > 0 {
		// Transfer balance to LikeChain ID
		balance = addrBalance

		// Remove key from db
		key := addr.DBKey("acc", "balance")
		state.MutableStateTree().Remove(key)
	} else {
		balance = state.GetInitialBalance()
	}

	// Initialize account info
	SaveBalance(state, id, balance)
	IncrementNextNonce(state, id)
}

func iterateLikeChainIDAddrPair(state context.IImmutableState, id *types.LikeChainID, fn func(id, addr []byte) bool) (isExist bool) {
	startingKey := getIDAddrPairPrefixKey(id)
	endingKey := append([]byte(nil), startingKey...)
	endingKey[len(endingKey)-1]++

	// Iterate the tree to check all addresses the given LikeChain ID has been bound
	return state.ImmutableStateTree().IterateRange(startingKey, endingKey, true, func(key, _ []byte) bool {
		if len(key) != len(startingKey)+20 {
			return false
		}
		// If fn returns true, iteration will be stopped
		return fn(key[4:24], key[30:50])
	})
}

// IsLikeChainIDRegistered checks whether the given LikeChain ID has registered or not
func IsLikeChainIDRegistered(state context.IImmutableState, id *types.LikeChainID) bool {
	idBytes := id.Bytes()
	return iterateLikeChainIDAddrPair(state, id, func(dbBytes, _ []byte) bool {
		return bytes.Compare(dbBytes, idBytes) == 0
	})
}

// IsAddressRegistered checks whether the given Address has registered or not
func IsAddressRegistered(state context.IImmutableState, addr *types.Address) bool {
	_, value := state.ImmutableStateTree().Get(getAddrIDPairKey(addr))
	return value != nil
}

// IsLikeChainIDHasAddress checks whether the given address has been bound to the given LikeChain ID
func IsLikeChainIDHasAddress(state context.IImmutableState, id *types.LikeChainID, addr *types.Address) bool {
	_, value := state.ImmutableStateTree().Get(getIDAddrPairKey(id, addr))
	return value != nil
}

var likeChainIDSeedKey = []byte("$account.likeChainIDSeed")

func generateLikeChainID(state context.IMutableState) *types.LikeChainID {
	var seedInt uint64
	_, seed := state.ImmutableStateTree().Get(likeChainIDSeedKey)
	if seed == nil {
		seedInt = 1
		seed = make([]byte, 8)
		binary.BigEndian.PutUint64(seed, seedInt)
	} else {
		seedInt = uint64(binary.BigEndian.Uint64(seed))
	}

	blockHash := state.GetBlockHash()

	// Concat the seed and the block's hash
	content := make([]byte, len(seed)+len(blockHash))
	copy(content, seed)
	copy(content[len(seed):], blockHash)
	// Take first 20 bytes of Keccak256 hash to be LikeChainID
	content = crypto.Keccak256(content)[:20]

	// Increment and save seed
	seedInt++
	binary.BigEndian.PutUint64(seed, seedInt)
	state.MutableStateTree().Set(likeChainIDSeedKey, seed)

	result := types.LikeChainID{}
	copy(result[:], content)
	return &result
}

// AddressToLikeChainID gets LikeChain ID by Address
func AddressToLikeChainID(state context.IImmutableState, addr *types.Address) *types.LikeChainID {
	_, value := state.ImmutableStateTree().Get(getAddrIDPairKey(addr))
	if value != nil {
		return types.ID(value)
	}
	return nil
}

// IdentifierToLikeChainID converts a Identifier to LikeChain ID using address - LikeChain ID mapping
func IdentifierToLikeChainID(state context.IImmutableState, identifier types.Identifier) *types.LikeChainID {
	switch identifier.(type) {
	case *types.LikeChainID:
		id := identifier.(*types.LikeChainID)
		if IsLikeChainIDRegistered(state, id) {
			return id
		}
		return nil
	case *types.Address:
		addr := identifier.(*types.Address)
		return AddressToLikeChainID(state, addr)
	}
	return nil
}

// NormalizeIdentifier converts an identifier with an address to an identifier with LikeChain ID if the address has
// registered
func NormalizeIdentifier(state context.IImmutableState, identifier types.Identifier) types.Identifier {
	id := IdentifierToLikeChainID(state, identifier)
	if id != nil {
		return id
	}
	return identifier
}

// SaveBalance saves account balance by LikeChain ID
func SaveBalance(state context.IMutableState, identifier types.Identifier, balance *big.Int) {
	key := NormalizeIdentifier(state, identifier).DBKey("acc", "balance")
	state.MutableStateTree().Set(key, balance.Bytes())
}

// FetchBalance fetches account balance by normalized Identifier
func FetchBalance(state context.IImmutableState, identifier types.Identifier) *big.Int {
	return FetchRawBalance(state, NormalizeIdentifier(state, identifier))
}

// FetchRawBalance fetches account balance by Identifier
func FetchRawBalance(state context.IImmutableState, identifier types.Identifier) *big.Int {
	key := identifier.DBKey("acc", "balance")
	_, value := state.ImmutableStateTree().Get(key)
	balance := big.NewInt(0)
	balance = balance.SetBytes(value)
	return balance
}

// AddBalance adds account balance by Identifier
func AddBalance(state context.IMutableState, identifier types.Identifier, amount *big.Int) {
	balance := FetchBalance(state, identifier)
	balance.Add(balance, amount)
	SaveBalance(state, identifier, balance)
}

// MinusBalance minus account balance by Identifier
func MinusBalance(state context.IMutableState, identifier types.Identifier, amount *big.Int) {
	balance := FetchBalance(state, identifier)
	balance.Sub(balance, amount)
	SaveBalance(state, identifier, balance)
}

// IncrementNextNonce increments next nonce of an account by LikeChain ID
// This also initialize next nonce of an account
func IncrementNextNonce(state context.IMutableState, id *types.LikeChainID) {
	nextNonceInt := FetchNextNonce(state, id) + 1
	nextNonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nextNonce, nextNonceInt)
	state.MutableStateTree().Set(id.DBKey("acc", "nextNonce"), nextNonce)
}

// FetchNextNonce fetches next nonce of an account by LikeChain ID
func FetchNextNonce(state context.IImmutableState, id *types.LikeChainID) uint64 {
	_, bytes := state.ImmutableStateTree().Get(id.DBKey("acc", "nextNonce"))
	if bytes == nil {
		return uint64(0)
	}
	return uint64(binary.BigEndian.Uint64(bytes))
}
