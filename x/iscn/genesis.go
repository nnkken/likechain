package iscn

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/likecoin/likechain/x/iscn/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func InitGenesis(ctx sdk.Context, keeper Keeper, genesisState GenesisState) []abci.ValidatorUpdate {
	keeper.SetParams(ctx, genesisState.Params)
	for _, cidData := range genesisState.CIDs {
		keeper.SetCidBlock(ctx, cidData.CID, cidData.Data)
	}
	for _, recordData := range genesisState.IscnRecords {
		keeper.SetIscnKernelRecord(ctx, recordData.ID, recordData.Record)
	}
	keeper.SetIscnCount(ctx, uint64(len(genesisState.IscnRecords)))
	return nil
}

func ExportGenesis(ctx sdk.Context, keeper Keeper) GenesisState {
	genesis := GenesisState{
		Params: keeper.GetParams(ctx),
	}
	keeper.IterateCidBlocks(ctx, func(cid CID, bz []byte) bool {
		genesis.CIDs = append(genesis.CIDs, types.CidData{
			CID:  cid,
			Data: bz,
		})
		return false
	})
	keeper.IterateIscnKernelRecords(ctx, func(id IscnID, record KernelRecord) bool {
		genesis.IscnRecords = append(genesis.IscnRecords, types.IscnRecordExported{
			ID:     id,
			Record: record,
		})
		return false
	})
	return genesis
}
