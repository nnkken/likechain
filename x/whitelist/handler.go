package whitelist

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
)

func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		switch msg := msg.(type) {
		case MsgSetWhitelist:
			return handleMsgSetWhitelist(ctx, msg, keeper)
		default:
			errMsg := fmt.Sprintf("unrecognized whitelist message type: %T", msg)
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}
	}
}

func handleMsgSetWhitelist(ctx sdk.Context, msg MsgSetWhitelist, keeper Keeper) (*sdk.Result, error) {
	approver := keeper.Approver(ctx)
	if !approver.Equals(msg.Approver) {
		return nil, ErrInvalidApprover
	}
	keeper.SetWhitelist(ctx, msg.Whitelist)
	bz, err := json.Marshal(msg.Whitelist)
	if err != nil {
		panic(err)
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			EventTypeSetWhitelist,
			sdk.NewAttribute(AttributeKeyWhitelist, string(bz)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Approver.String()),
		),
	})

	return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}

func WrapStakingHandler(keeper Keeper, stakingHandler sdk.Handler) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case staking.MsgCreateValidator:
			err := checkWhitelist(ctx, keeper, msg)
			if err != nil {
				return nil, err
			}
		}
		return stakingHandler(ctx, msg)
	}
}

func checkWhitelist(ctx sdk.Context, keeper Keeper, msg staking.MsgCreateValidator) error {
	whitelist := keeper.GetWhitelist(ctx)
	if len(whitelist) > 0 {
		for _, v := range whitelist {
			if msg.ValidatorAddress.Equals(v) {
				return nil
			}
		}
		return ErrValidatorNotInWEhitelist
	}
	return nil
}
