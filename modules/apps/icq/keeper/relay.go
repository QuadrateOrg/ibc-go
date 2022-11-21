package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v6/modules/apps/icq/types"
	icqtypes "github.com/cosmos/ibc-go/v6/modules/apps/icq/types"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (k Keeper) SendQuery(
	ctx sdk.Context,
	sourcePort,
	sourceChannel string,
	chanCap *capabilitytypes.Capability,
	req abci.RequestQuery,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
) (uint64, error) {
	if !k.IsControllerEnabled(ctx) {
		return 0, types.ErrControllerDisabled
	}

	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, sdkerrors.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	icqPacketData := types.InterchainQueryPacketData{
		Request: req,
	}

	return k.createOutgoingPacket(ctx, sourcePort, sourceChannel, destinationPort, destinationChannel, chanCap, icqPacketData, timeoutTimestamp)
}

func (k Keeper) createOutgoingPacket(
	ctx sdk.Context,
	sourcePort,
	sourceChannel,
	destinationPort,
	destinationChannel string,
	chanCap *capabilitytypes.Capability,
	icqPacketData types.InterchainQueryPacketData,
	timeoutTimestamp uint64,
) (uint64, error) {
	if err := icqPacketData.ValidateBasic(); err != nil {
		return 0, sdkerrors.Wrap(err, "invalid interchain query packet data")
	}

	return k.ics4Wrapper.SendPacket(
		ctx,
		chanCap,
		sourcePort,
		sourceChannel,
		clienttypes.ZeroHeight(),
		timeoutTimestamp,
		icqPacketData.GetBytes(),
	)
}

// OnRecvPacket handles a given interchain queries packet on a destination host chain.
// If the transaction is successfully executed, the transaction response bytes will be returned.
func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet) ([]byte, error) {
	var data icqtypes.InterchainQueryPacketData

	if err := icqtypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		// UnmarshalJSON errors are indeterminate and therefore are not wrapped and included in failed acks
		return nil, sdkerrors.Wrapf(icqtypes.ErrUnknownDataType, "cannot unmarshal ICQ packet data")
	}

	response, err := k.executeQuery(ctx, data.Request)
	if err != nil {
		return nil, err
	}
	return response, err
}

func (k Keeper) executeQuery(ctx sdk.Context, q abci.RequestQuery) ([]byte, error) {
	if err := k.authenticateQuery(ctx, q); err != nil {
		return nil, err
	}

	response := k.querier.Query(q)
	// Remove non-deterministic fields from response
	response = abci.ResponseQuery{
		Code:     response.Code,
		Index:    response.Index,
		Key:      response.Key,
		Value:    response.Value,
		ProofOps: response.ProofOps,
		Height:   response.Height,
	}

	ack := icqtypes.InterchainQueryPacketAck{
		Response: response,
	}
	data, err := icqtypes.ModuleCdc.MarshalJSON(&ack)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to marshal tx data")
	}

	return data, nil
}

// authenticateQuery ensures the provided query request is in the whitelist.
func (k Keeper) authenticateQuery(ctx sdk.Context, q abci.RequestQuery) error {
	allowQueries := k.GetAllowQueries(ctx)
	if !types.ContainsQueryPath(allowQueries, q.Path) {
		return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "query path not allowed: %s", q.Path)
	}
	if !(q.Height == 0 || q.Height == ctx.BlockHeight()) {
		return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "query height not allowed: %d", q.Height)
	}
	if q.Prove {
		return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "query proof not allowed")
	}

	return nil
}
