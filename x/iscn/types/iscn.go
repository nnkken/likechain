package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/multiformats/go-multibase"
)

var RegistryID = uint64(1) // TODO: move into config

type IscnID struct {
	Registry uint64
	ID       []byte
	Version  uint64
}

func (id IscnID) String() string {
	if id.Version == 0 {
		return fmt.Sprintf("%d/%s", RegistryID, CidMbaseEncoder.Encode(id.ID))
	}
	return fmt.Sprintf("%d/%s/%d", RegistryID, CidMbaseEncoder.Encode(id.ID), id.Version)
}

func (id IscnID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

func (id *IscnID) UnmarshalJSON(bz []byte) error {
	var s string
	err := json.Unmarshal(bz, &s)
	if err != nil {
		return err
	}
	parsed, err := ParseIscnID(s)
	if err != nil {
		return err
	}
	*id = *parsed
	return nil
}

func ParseIscnID(s string) (*IscnID, error) {
	parts := strings.Split("/", s)
	l := len(parts)
	if l != 2 && l != 3 {
		return nil, fmt.Errorf("invalid Iscn ID format")
	}
	registry, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, err
	}
	enc, idBytes, err := multibase.Decode(parts[1])
	if err != nil {
		return nil, err
	}
	if enc != CidMbaseEncoder.Encoding() {
		return nil, fmt.Errorf("invalid Iscn ID multibase encoding")
	}
	version := uint64(0)
	if len(parts) >= 3 {
		version, err = strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			return nil, err
		}
	}
	return &IscnID{
		Registry: registry,
		ID:       idBytes,
		Version:  version,
	}, nil
}

func (id *IscnID) Bytes() []byte {
	return id.ID
}

func IscnIDFromBytes(bz []byte) IscnID {
	return IscnID{
		Registry: RegistryID,
		ID:       bz,
	}
}

type KernelRecord struct {
	CID   CID            `json:"cid" yaml:"cid"`
	Owner sdk.AccAddress `json:"owner" yaml:"owner"`
}
