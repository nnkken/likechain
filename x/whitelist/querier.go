package whitelist

import (
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func NewQuerier(k Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err error) {
		switch path[0] {
		case QueryApprover:
			return queryApprover(ctx, req, k)
		case QueryWhitelist:
			return queryWhitelist(ctx, req, k)
		default:
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "unknown whitelist query endpoint")
		}
	}
}

func queryApprover(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	approver := k.Approver(ctx)

	res, err := codec.MarshalJSONIndent(ModuleCdc, approver)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}

func queryWhitelist(ctx sdk.Context, req abci.RequestQuery, k Keeper) ([]byte, error) {
	whitelist := k.GetWhitelist(ctx)

	res, err := codec.MarshalJSONIndent(ModuleCdc, whitelist)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}
