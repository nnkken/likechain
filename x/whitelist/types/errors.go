package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

var (
	ErrInvalidApprover          = sdkerrors.Register(types.ModuleName, 101, "approver address is invalid")
	ErrValidatorNotInWEhitelist = sdkerrors.Register(types.ModuleName, 102, "validator not in whitelist")
)
