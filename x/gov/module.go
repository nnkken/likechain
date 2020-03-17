package gov

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/staking/exported"
)

var (
	_ module.AppModule = AppModule{}
)

type AppModule struct {
	gov.AppModule
	stakingKeeper types.StakingKeeper
}

func NewAppModule(keeper gov.Keeper, accountKeeper types.AccountKeeper, supplyKeeper types.SupplyKeeper, stakingKeeper types.StakingKeeper) AppModule {
	return AppModule{
		AppModule:     gov.NewAppModule(keeper, accountKeeper, supplyKeeper),
		stakingKeeper: stakingKeeper,
	}
}

func checkIsValidator(ctx sdk.Context, stakingKeeper types.StakingKeeper, addr sdk.Address) bool {
	isValidator := false
	stakingKeeper.IterateBondedValidatorsByPower(ctx, func(index int64, validator exported.ValidatorI) (stop bool) {
		if validator.GetOperator().Equals(addr) {
			isValidator = true
			return true
		}
		return false
	})
	return isValidator
}

// proxy handler which intercepts MsgSubmitProposal
func (am AppModule) NewHandler() sdk.Handler {
	govHandler := am.AppModule.NewHandler()
	wrappedHandler := func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case gov.MsgSubmitProposal:
			if !checkIsValidator(ctx, am.stakingKeeper, msg.Proposer) {
				errMsg := fmt.Sprintf("only validators can submit proposals")
				return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, errMsg)
			}
		case gov.MsgVote:
			if !checkIsValidator(ctx, am.stakingKeeper, msg.Voter) {
				errMsg := fmt.Sprintf("only validators can vote")
				return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, errMsg)
			}
		}
		return govHandler(ctx, msg)
	}
	return wrappedHandler
}
