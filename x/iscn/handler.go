package iscn

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cbornode "github.com/ipfs/go-ipld-cbor"
)

func NewHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case MsgCreateIscn:
			return handleMsgCreateIscn(ctx, msg, keeper)
		case MsgAddEntity:
			return handleMsgAddEntity(ctx, msg, keeper)
		default:
			errMsg := fmt.Sprintf("unrecognized iscn message type: %T", msg)
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleEntity(ctx sdk.Context, entity IscnDataField, keeper Keeper) (*CID, error) {
	switch entity.Type() {
	case NestedIscnData:
		e, _ := entity.AsIscnData()
		return keeper.SetEntity(ctx, e)
	case NestedCID:
		cid, _ := entity.AsCID()
		if keeper.GetEntity(ctx, *cid) == nil {
			return nil, fmt.Errorf("unknown entity CID: %s", cid)
		}
		return cid, nil
	default:
		return nil, fmt.Errorf("entity does not match schema")
	}
}

func handleRightTerms(ctx sdk.Context, rightTerms IscnDataField, keeper Keeper) (*CID, error) {
	switch rightTerms.Type() {
	case NestedCID:
		cid, _ := rightTerms.AsCID()
		return cid, nil
	default:
		return nil, fmt.Errorf("right terms does not match schema")
	}
}

func handleStakeholders(ctx sdk.Context, stakeholders IscnData, keeper Keeper) (*CID, error) {
	stakeholdersArr, _ := stakeholders.Get("stakeholders").AsArray()
	for i := 0; i < stakeholdersArr.Len(); i++ {
		stakeholder, _ := stakeholdersArr.Get(i).AsIscnData()
		entityField := stakeholder.Get("stakeholder")
		cid, err := handleEntity(ctx, entityField, keeper)
		if err != nil {
			return nil, err
		}
		stakeholder.Set("stakeholder", *cid)
		n, _ := stakeholder.Get("sharing").AsUint64()
		stakeholder.Set("sharing", uint32(n))
	}
	schemaVersion := uint64(1)
	return keeper.SetCidIscnObject(ctx, stakeholders, StakeholdersCodecType, schemaVersion)
}

func handleRights(ctx sdk.Context, rights IscnData, keeper Keeper) (*CID, error) {
	rightsArr, _ := rights.Get("rights").AsArray()
	for i := 0; i < rightsArr.Len(); i++ {
		right, _ := rightsArr.Get(i).AsIscnData()
		holderField := right.Get("holder")
		cid, err := handleEntity(ctx, holderField, keeper)
		if err != nil {
			return nil, err
		}
		right.Set("holder", *cid)

		termsField := right.Get("terms")
		cid, err = handleRightTerms(ctx, termsField, keeper)
		if err != nil {
			return nil, err
		}
		right.Set("terms", *cid)
	}
	schemaVersion := uint64(1)
	return keeper.SetCidIscnObject(ctx, rights, RightsCodecType, schemaVersion)
}

func handleIscnContent(ctx sdk.Context, content IscnDataField, keeper Keeper) (*CID, error) {
	switch content.Type() {
	case NestedIscnData:
		content, _ := content.AsIscnData()
		version, _ := content.Get("version").AsUint64()
		content.Set("version", version)
		parentCID, ok := content.Get("parent").AsCID()
		if ok {
			parent := keeper.GetEntity(ctx, *parentCID)
			if keeper.GetEntity(ctx, *parentCID) == nil {
				return nil, fmt.Errorf("unknown ISCN content parent CID: %s", parentCID)
			}
			parentVersion, err := parent.GetUint64("version")
			if err != nil || parentVersion != version-1 {
				return nil, fmt.Errorf("invalid content version: %d", version)
			}
			content.Set("parent", *parentCID)
		} else if version != 1 {
			return nil, fmt.Errorf("invalid content version, expect 1, got %d", version)
		}
		return keeper.SetIscnContent(ctx, content)
	case NestedCID:
		cid, _ := content.AsCID()
		if keeper.GetEntity(ctx, *cid) == nil {
			return nil, fmt.Errorf("unknown ISCN content CID: %s", cid)
		}
		return cid, nil
	default:
		return nil, fmt.Errorf("ISCN content does not match schema")
	}
}

func handleMsgCreateIscn(ctx sdk.Context, msg MsgCreateIscn, keeper Keeper) sdk.Result {
	kernelRawMap := RawIscnMap{}
	err := cbornode.DecodeInto(msg.IscnKernel, &kernelRawMap)
	if err != nil {
		return sdk.Result{
			/* TODO: proper error*/
			Code:      123,
			Codespace: DefaultCodespace,
			Log:       fmt.Sprintf("unable to decode ISCN kernel data: %s", err.Error()),
		}
	}
	kernel, ok := KernelSchema.ConstructIscnData(kernelRawMap)
	if !ok {
		return sdk.Result{
			/* TODO: proper error*/
			Code:      123,
			Codespace: DefaultCodespace,
			Log:       "ISCN kernel does not fulfill schema",
		}
	}
	err = keeper.DeductFeeForIscn(ctx, msg.From, msg.IscnKernel)
	if err != nil {
		return sdk.Result{
			/* TODO: proper error*/
			Code:      123,
			Codespace: DefaultCodespace,
			Log:       err.Error(),
		}
	}
	stakeholders, _ := kernel.Get("stakeholders").AsIscnData()
	stakeholdersCID, err := handleStakeholders(ctx, stakeholders, keeper)
	if err != nil {
		return sdk.Result{
			Code:      123, // TODO
			Codespace: DefaultCodespace,
			Log:       err.Error(),
		}
	}
	kernel.Set("stakeholders", *stakeholdersCID)
	rights, _ := kernel.Get("rights").AsIscnData()
	rightsCID, err := handleRights(ctx, rights, keeper)
	if err != nil {
		return sdk.Result{
			Code:      123, // TODO
			Codespace: DefaultCodespace,
			Log:       err.Error(),
		}
	}
	kernel.Set("rights", *rightsCID)
	content := kernel.Get("content")
	contentCID, err := handleIscnContent(ctx, content, keeper)
	if err != nil {
		return sdk.Result{
			Code:      123, // TODO
			Codespace: DefaultCodespace,
			Log:       err.Error(),
		}
	}
	kernel.Set("content", *contentCID)
	t, _ := kernel.Get("timestamp").AsTime()
	if t.After(ctx.BlockHeader().Time) {
		return sdk.Result{
			Code:      123, // TODO
			Codespace: DefaultCodespace,
			Log:       fmt.Sprintf("kernel time is after blocktime"),
		}
	}
	version, _ := kernel.Get("version").AsUint64()
	kernel.Set("version", version)
	parent := kernel.Get("parent")
	switch parent.Type() {
	case None:
		// New ISCN
		// nil parent case version checking should be handled by ValidateBasic
		_, err = keeper.AddIscnKernel(ctx, msg.From, kernel)
		if err != nil {
			return sdk.Result{
				Code:      123, // TODO
				Codespace: DefaultCodespace,
				Log:       err.Error(),
			}
		}
	case NestedCID:
		// Old ISCN
		// TODO: check if the content's parent is pointing to content with the same ISCN ID
		// seems complicated checkings for different weird cases
		parentKernelCID, _ := parent.AsCID()
		parentKernelObj := keeper.GetIscnKernelByCID(ctx, *parentKernelCID)
		if parentKernelObj == nil {
			return sdk.Result{
				Code:      123, // TODO
				Codespace: DefaultCodespace,
				Log:       fmt.Sprintf("unknown parent ISCN kernel CID: %s", parentKernelCID),
			}
		}
		kernel.Set("parent", *parentKernelCID)
		iscnIDBytes, _ := parentKernelObj.GetBytes("id")
		iscnID := IscnIDFromBytes(iscnIDBytes)
		record := keeper.GetIscnKernelRecord(ctx, iscnID)
		if record == nil {
			// TODO: bug, should log or panic?
			return sdk.Result{
				Code:      123, // TODO
				Codespace: DefaultCodespace,
				Log:       fmt.Sprintf("unknown parent ISCN kernel CID: %s", parentKernelCID),
			}
		}
		if !record.Owner.Equals(msg.From) {
			return sdk.Result{
				Code:      123, // TODO
				Codespace: DefaultCodespace,
				Log:       "sender is not the owner of the parent ISCN record",
			}
		}
		// TODO: check ID in incoming kernel record
		parentVersion, _ := parentKernelObj.GetUint64("version")
		if version != parentVersion+1 {
			return sdk.Result{
				Code:      123, // TODO
				Codespace: DefaultCodespace,
				Log:       "invalid ISCN kernel version",
			}
		}
		cid, err := keeper.SetIscnKernel(ctx, iscnID, kernel)
		if err != nil {
			return sdk.Result{
				Code:      123, // TODO
				Codespace: DefaultCodespace,
				Log:       err.Error(),
			}
		}
		record.CID = *cid
		keeper.SetIscnKernelRecord(ctx, iscnID, *record)
	default:
		return sdk.Result{
			Code:      123, // TODO
			Codespace: DefaultCodespace,
			Log:       "ISCN kernel parent does not fulfill schema",
		}
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.From.String()),
		),
	)

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func handleMsgAddEntity(ctx sdk.Context, msg MsgAddEntity, keeper Keeper) sdk.Result {
	entityRawMap := RawIscnMap{}
	err := cbornode.DecodeInto(msg.Entity, &entityRawMap)
	if err != nil {
		return sdk.Result{
			/* TODO: proper error*/
			Code:      123,
			Codespace: DefaultCodespace,
			Log:       fmt.Sprintf("unable to decode entity data: %s", err.Error()),
		}
	}
	entity, ok := EntitySchema.ConstructIscnData(entityRawMap)
	if !ok {
		return sdk.Result{
			/* TODO: proper error*/
			Code:      123,
			Codespace: DefaultCodespace,
			Log:       "entity does not fulfill schema",
		}
	}
	err = keeper.DeductFeeForIscn(ctx, msg.From, msg.Entity) // TODO: different fee for entity
	if err != nil {
		return sdk.Result{
			/* TODO: proper error*/
			Code:      123,
			Codespace: DefaultCodespace,
			Log:       err.Error(),
		}
	}
	_, err = keeper.SetEntity(ctx, entity)
	if err != nil {
		return sdk.Result{
			Code:      123, // TODO
			Codespace: DefaultCodespace,
			Log:       err.Error(),
		}
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.From.String()),
		),
	)
	return sdk.Result{Events: ctx.EventManager().Events()}
}
