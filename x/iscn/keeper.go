package iscn

import (
	"encoding/binary"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/tendermint/tendermint/crypto/tmhash"

	gocid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"

	iscnblock "github.com/likecoin/iscn-ipld/plugin/block"
)

const (
	DefaultParamspace = ModuleName
)

type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) auth.Account
}

type Keeper struct {
	storeKey      sdk.StoreKey
	cdc           *codec.Codec
	paramstore    params.Subspace
	codespace     sdk.CodespaceType
	accountKeeper AccountKeeper
	supplyKeeper  authTypes.SupplyKeeper
}

func NewKeeper(
	cdc *codec.Codec, key sdk.StoreKey, accountKeeper AccountKeeper,
	supplyKeeper authTypes.SupplyKeeper, paramstore params.Subspace,
	codespace sdk.CodespaceType,
) Keeper {
	return Keeper{
		storeKey:      key,
		cdc:           cdc,
		paramstore:    paramstore.WithKeyTable(ParamKeyTable()),
		codespace:     codespace,
		accountKeeper: accountKeeper,
		supplyKeeper:  supplyKeeper,
	}
}

func (k Keeper) Codespace() sdk.CodespaceType {
	return k.codespace
}

func ParamKeyTable() params.KeyTable {
	return params.NewKeyTable().RegisterParamSet(&Params{})
}

func (k Keeper) FeePerByte(ctx sdk.Context) (res sdk.DecCoin) {
	k.paramstore.Get(ctx, KeyFeePerByte, &res)
	return
}

func (k Keeper) GetParams(ctx sdk.Context) Params {
	return Params{
		FeePerByte: k.FeePerByte(ctx),
	}
}

func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

func (k Keeper) GetCidBlock(ctx sdk.Context, cid CID) []byte {
	key := GetCidBlockKey(cid.Bytes())
	return ctx.KVStore(k.storeKey).Get(key)
}

func (k Keeper) HasCidBlock(ctx sdk.Context, cid CID) bool {
	key := GetCidBlockKey(cid.Bytes())
	return ctx.KVStore(k.storeKey).Has(key)
}

func (k Keeper) SetCidBlock(ctx sdk.Context, cid CID, bz []byte) {
	key := GetCidBlockKey(cid.Bytes())
	ctx.KVStore(k.storeKey).Set(key, bz)
}

func (k Keeper) GetCidIscnObject(ctx sdk.Context, cid CID) iscnblock.IscnObject {
	bz := k.GetCidBlock(ctx, cid)
	if bz == nil {
		return nil
	}
	// IscnObject flow
	obj, err := iscnblock.Decode(bz, cid)
	if err != nil {
		return nil
	}
	return obj
}

func (k Keeper) SetCidIscnObject(
	ctx sdk.Context, data IscnData,
	codec uint64, schemaVersion uint64,
) (*CID, error) {
	obj, err := iscnblock.Encode(codec, schemaVersion, data)
	if err != nil {
		return nil, err
	}
	bz := obj.RawData()
	cid := obj.Cid()

	k.SetCidBlock(ctx, cid, bz)
	return &cid, nil
}

func (k Keeper) checkCodecAndGetIscnObject(ctx sdk.Context, cid CID, codec uint64) iscnblock.IscnObject {
	if cid.Prefix().GetCodec() != codec {
		return nil
	}
	return k.GetCidIscnObject(ctx, cid)
}

func (k Keeper) setIscnObjectAndEmitEvent(
	ctx sdk.Context, data IscnData, codec uint64, schemaVersion uint64,
	eventType string, attr string,
) (*CID, error) {
	cid, err := k.SetCidIscnObject(ctx, data, codec, schemaVersion)
	if err != nil {
		return nil, err
	}
	cidStr, err := cid.StringOfBase(CidMbaseEncoder.Encoding())
	if err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			eventType,
			sdk.NewAttribute(attr, cidStr),
		),
	)
	return cid, nil
}

func (k Keeper) GetEntity(ctx sdk.Context, cid CID) iscnblock.IscnObject {
	return k.checkCodecAndGetIscnObject(ctx, cid, EntityCodecType)
}

func (k Keeper) SetEntity(ctx sdk.Context, entity IscnData) (*CID, error) {
	// TODO: schemaVersion base on input context field
	schemaVersion := uint64(1)
	return k.setIscnObjectAndEmitEvent(
		ctx, entity, EntityCodecType, schemaVersion, EventTypeAddEntity, AttributeKeyEntityCid,
	)
}

func (k Keeper) GetRightTerms(ctx sdk.Context, cid CID) *string {
	bz := k.GetCidBlock(ctx, cid)
	if bz == nil {
		return nil
	}
	terms := string(bz)
	return &terms
}

func (k Keeper) SetRightTerms(ctx sdk.Context, terms string) (*CID, error) {
	bz := []byte(terms)
	cid, err := gocid.V1Builder{
		Codec:  RightTermsCodecType,
		MhType: mh.SHA2_256,
	}.Sum(bz)
	if err != nil {
		return nil, err
	}
	k.SetCidBlock(ctx, cid, bz)
	cidStr, err := cid.StringOfBase(CidMbaseEncoder.Encoding())
	if err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			EventTypeAddRightTerms,
			sdk.NewAttribute(AttributeKeyRightTermsCid, cidStr),
		),
	)
	return &cid, nil
}

func (k Keeper) GetIscnContent(ctx sdk.Context, cid CID) iscnblock.IscnObject {
	return k.checkCodecAndGetIscnObject(ctx, cid, IscnContentCodecType)
}

func (k Keeper) SetIscnContent(ctx sdk.Context, content IscnData) (*CID, error) {
	// TODO: schemaVersion base on input context field
	schemaVersion := uint64(1)
	return k.setIscnObjectAndEmitEvent(
		ctx, content, IscnContentCodecType, schemaVersion, EventTypeAddIscnContent, AttributeKeyIscnContentCid,
	)
}

func (k Keeper) GetIscnKernelByCID(ctx sdk.Context, cid CID) iscnblock.IscnObject {
	if cid.Prefix().GetCodec() != IscnKernelCodecType {
		return nil
	}
	return k.GetCidIscnObject(ctx, cid)
}

func (k Keeper) GetIscnKernelCIDByIscnID(ctx sdk.Context, iscnID IscnID) *CID {
	key := GetIscnKernelKey(iscnID.Bytes())
	cidBytes := ctx.KVStore(k.storeKey).Get(key)
	if cidBytes == nil {
		return nil
	}
	_, cid, err := gocid.CidFromBytes(cidBytes)
	if err != nil {
		// TODO: should panic or at least log
		return nil
	}
	return &cid
}

func (k Keeper) SetIscnKernel(ctx sdk.Context, iscnID IscnID, kernel IscnData) (*CID, error) {
	// TODO: schemaVersion base on input context field
	schemaVersion := uint64(1)
	idBytes := iscnID.Bytes()
	kernel.Set("id", idBytes)
	cid, err := k.setIscnObjectAndEmitEvent(
		ctx, kernel, IscnKernelCodecType, schemaVersion, EventTypeAddIscnKernel, AttributeKeyIscnKernelCid,
	)
	if err != nil {
		return nil, err
	}
	cidBytes := cid.Bytes()
	key := GetIscnKernelKey(idBytes)
	ctx.KVStore(k.storeKey).Set(key, cidBytes)
	key = GetCidToIscnIDKey(cidBytes)
	ctx.KVStore(k.storeKey).Set(key, idBytes)
	return cid, err
}

func (k Keeper) SetIscnOwner(ctx sdk.Context, iscnID IscnID, owner sdk.AccAddress) {
	key := GetIscnOwnerKey(iscnID.Bytes())
	ctx.KVStore(k.storeKey).Set(key, owner)
}

func (k Keeper) GetIscnOwner(ctx sdk.Context, iscnID IscnID) sdk.AccAddress {
	key := GetIscnOwnerKey(iscnID.Bytes())
	bz := ctx.KVStore(k.storeKey).Get(key)
	if bz == nil {
		return nil
	}
	return sdk.AccAddress(bz)
}

func (k Keeper) SetIscnCount(ctx sdk.Context, count uint64) {
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(count)
	ctx.KVStore(k.storeKey).Set(IscnCountKey, bz)
}

func (k Keeper) GetIscnCount(ctx sdk.Context) uint64 {
	count := uint64(0)
	bz := ctx.KVStore(k.storeKey).Get(IscnCountKey)
	if bz != nil {
		k.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &count)
	}
	return count
}

func (k Keeper) DeductFeeForIscn(ctx sdk.Context, feePayer sdk.AccAddress, data []byte) error {
	acc := k.accountKeeper.GetAccount(ctx, feePayer)
	if acc == nil {
		return fmt.Errorf("No account") // TODO: proper error
	}
	feePerByte := k.GetParams(ctx).FeePerByte
	feeAmount := feePerByte.Amount.MulInt64(int64(len(data)))
	fees := sdk.NewCoins(sdk.NewCoin(feePerByte.Denom, feeAmount.Ceil().RoundInt()))
	result := auth.DeductFees(k.supplyKeeper, ctx, acc, fees)
	if !result.IsOK() {
		// TODO: proper error
		return fmt.Errorf("Not enough fee, %s %s needed", feeAmount.String(), feePerByte.Denom)
	}
	return nil
}

func (k Keeper) AddIscnKernel(
	ctx sdk.Context, owner sdk.AccAddress, kernel IscnData,
) (iscnID IscnID, err error) {
	hasher := tmhash.New()
	hasher.Write(ctx.BlockHeader().LastBlockId.Hash)
	iscnCount := k.GetIscnCount(ctx)
	k.SetIscnCount(ctx, iscnCount+1)
	binary.Write(hasher, binary.BigEndian, iscnCount)
	idBytes := hasher.Sum(nil)
	iscnID = IscnIDFromBytes(idBytes)
	_, err = k.SetIscnKernel(ctx, iscnID, kernel)
	if err != nil {
		return iscnID, err
	}
	k.SetIscnOwner(ctx, iscnID, owner)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			EventTypeCreateIscn,
			sdk.NewAttribute(AttributeKeyIscnID, iscnID.String()),
		),
	)
	return iscnID, nil
}
