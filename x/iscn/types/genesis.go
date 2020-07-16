package types

import (
	"fmt"
)

type CidData struct {
	CID  CID    `json:"cid" yaml:"cid"`
	Data []byte `json:"data" yaml:"data"`
}

type IscnRecordExported struct {
	ID     IscnID       `json:"id" yaml:"id"`
	Record KernelRecord `json:"record" yaml:"record"`
}

type GenesisState struct {
	Params      Params               `json:"params" yaml:"params"`
	CIDs        []CidData            `json:"cids" yaml:"cids"`
	IscnRecords []IscnRecordExported `json:"iscn_records" yaml:"iscn_records"`
}

func DefaultGenesisState() GenesisState {
	return GenesisState{
		Params: DefaultParams(),
	}
}

func ValidateGenesis(data GenesisState) error {
	usedCIDs := map[string]bool{}
	for _, cidData := range data.CIDs {
		cid := cidData.CID
		normalizedCID := cid.String()
		if usedCIDs[normalizedCID] {
			return fmt.Errorf("Repeated CID: %s", normalizedCID)
		}
		usedCIDs[normalizedCID] = true
	}
	usedIscnIDs := map[string]bool{}
	for _, recordData := range data.IscnRecords {
		id := recordData.ID
		idStr := id.String()
		if id.Version != 0 {
			return fmt.Errorf("Iscn ID has non-zero version: %s", idStr)
		}
		if usedIscnIDs[idStr] {
			return fmt.Errorf("Repeated Iscn ID: %s", idStr)
		}
		usedCIDs[idStr] = true
	}
	return nil
}
