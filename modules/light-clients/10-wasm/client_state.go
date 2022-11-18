package wasm

import (
	"encoding/json"
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v5/modules/core/exported"
)

var _ exported.ClientState = (*ClientState)(nil)

func (c *ClientState) ClientType() string {
	return exported.Wasm
}

func (c *ClientState) GetLatestHeight() exported.Height {
	return c.LatestHeight
}

func (c *ClientState) Validate() error {
	if c.Data == nil || len(c.Data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	if c.CodeId == nil || len(c.CodeId) == 0 {
		return fmt.Errorf("codeid cannot be empty")
	}
	return nil
}

func (c *ClientState) Status(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec) exported.Status {
	// TODO: store the status of the client in the SDK store to make it easier to query?
	return exported.Active
}

func (c *ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	const ExportMetadataQuery = "exportmetadata"
	payload := make(map[string]map[string]interface{})
	payload[ExportMetadataQuery] = make(map[string]interface{})
	inner := payload[ExportMetadataQuery]
	inner["me"] = c

	encodedData, err := json.Marshal(payload)
	if err != nil {
		// TODO: Handle error
	}
	response, err := queryContractWithStore(c.CodeId, store, encodedData)
	if err != nil {
		// TODO: Handle error
	}

	output := queryResponse{}
	if err := json.Unmarshal(response, &output); err != nil {
		// TODO: Handle error
	}

	genesisMetadata := make([]exported.GenesisMetadata, len(output.GenesisMetadata))
	for i, metadata := range output.GenesisMetadata {
		genesisMetadata[i] = metadata
	}
	return genesisMetadata
}

func (c *ClientState) ZeroCustomFields() exported.ClientState {
	const ZeroCustomFields = "zerocustomfields"
	payload := make(map[string]map[string]interface{})
	payload[ZeroCustomFields] = make(map[string]interface{})
	inner := payload[ZeroCustomFields]
	inner["me"] = c

	encodedData, err := json.Marshal(payload)
	if err != nil {
		// TODO: Handle error
	}

	gasMeter := sdk.NewGasMeter(maxGasLimit)
	mockEnv := wasmtypes.Env{
		Block: wasmtypes.BlockInfo{
			Height:  123,
			Time:    1578939743_987654321,
			ChainID: "foobar",
		},
		Transaction: &wasmtypes.TransactionInfo{
			Index: 4,
		},
		Contract: wasmtypes.ContractInfo{
			Address: "contract",
		},
	}
	out, err := callContractWithEnvAndMeter(c.CodeId, nil, &FailKVStore{}, mockEnv, gasMeter, encodedData)
	if err != nil {
		// TODO: Handle error
	}
	output := clientStateCallResponse{}
	if err := json.Unmarshal(out.Data, &output); err != nil {
		// TODO: Handle error
	}
	output.resetImmutables(c)
	return output.Me
}

func (c *ClientState) GetTimestampAtHeight(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
) (uint64, error) {
	// get consensus state at height from clientStore to check for expiry
	consState, found := GetConsensusState(clientStore, cdc, height)
	if found != nil {
		return 0, sdkerrors.Wrapf(clienttypes.ErrConsensusStateNotFound, "height (%s)", height)
	}
	return consState.GetTimestamp(), nil
}

func (c *ClientState) Initialize(context sdk.Context, marshaler codec.BinaryCodec, store sdk.KVStore, state exported.ConsensusState) error {
	const InitializeState = "initialize_state"
	payload := make(map[string]map[string]interface{})
	payload[InitializeState] = make(map[string]interface{})
	inner := payload[InitializeState]
	inner["me"] = c
	inner["consensus_state"] = state

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return sdkerrors.Wrapf(ErrUnableToMarshalPayload, fmt.Sprintf("underlying error: %s", err.Error()))
	}

	// Under the hood there are two calls to wasm contract for initialization as by design
	// cosmwasm does not allow init call to return any value.

	_, err = initContract(c.CodeId, context, store, encodedData)
	if err != nil {
		return sdkerrors.Wrapf(ErrUnableToInit, fmt.Sprintf("underlying error: %s", err.Error()))
	}

	out, err := callContract(c.CodeId, context, store, encodedData)
	if err != nil {
		return sdkerrors.Wrapf(ErrUnableToCall, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	output := clientStateCallResponse{}
	if err := json.Unmarshal(out.Data, &output); err != nil {
		return sdkerrors.Wrapf(ErrUnableToUnmarshalPayload, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	if !output.Result.IsValid {
		return fmt.Errorf("%s error occurred while initializing client state", output.Result.ErrorMsg)
	}
	output.resetImmutables(c)

	*c = *output.Me

	_, err = SaveClientStateIntoWasmStorage(context, marshaler, store, c)
	if err != nil {
		return fmt.Errorf("%s error occurred while duplicating into wasm storage", err.Error())
	}
	return nil
}

func (c *ClientState) VerifyMembership(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path []byte,
	value []byte,
) error {
	panic("implement me")
}

func (c *ClientState) VerifyNonMembership(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	delayTimePeriod uint64,
	delayBlockPeriod uint64,
	proof []byte,
	path []byte,
) error {
	panic("implement me")
}

// VerifyClientMessage must verify a ClientMessage. A ClientMessage could be a Header, Misbehaviour, or batch update.
// It must handle each type of ClientMessage appropriately. Calls to CheckForMisbehaviour, UpdateState, and UpdateStateOnMisbehaviour
// will assume that the content of the ClientMessage has been verified and can be trusted. An error should be returned
// if the ClientMessage fails to verify.
func (c *ClientState) VerifyClientMessage(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) error {
	return nil
}

func (c *ClientState) CheckForMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, msg exported.ClientMessage) bool {
	// TODO: implement
	return false
}

// UpdateStateOnMisbehaviour should perform appropriate state changes on a client state given that misbehaviour has been detected and verified
func (c *ClientState) UpdateStateOnMisbehaviour(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) {
	return
}

func (c *ClientState) UpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, clientMsg exported.ClientMessage) []exported.Height {
	_, err := SaveClientStateIntoWasmStorage(ctx, cdc, clientStore, c)
	// TODO: implement
	if err != nil {
		return []exported.Height{}
	}
	return []exported.Height{}
}

func (c *ClientState) CheckSubstituteAndUpdateState(
	ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore,
	substituteClientStore sdk.KVStore, substituteClient exported.ClientState,
) error {
	var (
		SubjectPrefix    = []byte("subject/")
		SubstitutePrefix = []byte("substitute/")
	)

	consensusState, err := GetConsensusState(subjectClientStore, cdc, c.LatestHeight)
	if err != nil {
		return sdkerrors.Wrapf(
			err, "unexpected error: could not get consensus state from clientstore at height: %d", c.GetLatestHeight(),
		)
	}

	store := NewWrappedStore(subjectClientStore, subjectClientStore, SubjectPrefix, SubstitutePrefix)

	const CheckSubstituteAndUpdateState = "checksubstituteandupdatestate"
	payload := make(map[string]map[string]interface{})
	payload[CheckSubstituteAndUpdateState] = make(map[string]interface{})
	inner := payload[CheckSubstituteAndUpdateState]
	inner["me"] = c
	inner["subject_consensus_state"] = consensusState
	inner["substitute_client_state"] = substituteClient
	// inner["initial_height"] = initialHeight

	encodedData, err := json.Marshal(payload)
	if err != nil {
		return sdkerrors.Wrapf(ErrUnableToMarshalPayload, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	out, err := callContract(c.CodeId, ctx, store, encodedData)
	if err != nil {
		return sdkerrors.Wrapf(ErrUnableToCall, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	output := clientStateCallResponse{}
	if err := json.Unmarshal(out.Data, &output); err != nil {
		return sdkerrors.Wrapf(ErrUnableToUnmarshalPayload, fmt.Sprintf("underlying error: %s", err.Error()))
	}
	if !output.Result.IsValid {
		return fmt.Errorf("%s error occurred while updating client state", output.Result.ErrorMsg)
	}

	output.resetImmutables(c)
	return nil
}
func (c *ClientState) VerifyUpgradeAndUpdateState(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	store sdk.KVStore,
	newClient exported.ClientState,
	newConsState exported.ConsensusState,
	proofUpgradeClient,
	proofUpgradeConsState []byte,
) error {
	_, err := SaveClientStateIntoWasmStorage(ctx, cdc, store, c)
	// TODO: implement
	if err != nil {
		return fmt.Errorf("%s error occurred while duplicating into wasm storage", err.Error())
	}
	return nil
}
