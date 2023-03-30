package ibctesting

import (
	"time"

	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
	"github.com/cosmos/ibc-go/v7/testing/mock"
)

type ClientConfig interface {
	GetClientType() string
}

type TendermintConfig struct {
	TrustLevel      ibctm.Fraction
	TrustingPeriod  time.Duration
	UnbondingPeriod time.Duration
	MaxClockDrift   time.Duration
	Wasm            bool
}

func NewTendermintConfig(wasm bool) *TendermintConfig {
	return &TendermintConfig{
		TrustLevel:      DefaultTrustLevel,
		TrustingPeriod:  TrustingPeriod,
		UnbondingPeriod: UnbondingPeriod,
		MaxClockDrift:   MaxClockDrift,
		Wasm:            wasm,
	}
}

func (tmcfg *TendermintConfig) GetClientType() string {
	if tmcfg.Wasm {
		return exported.Wasm
	}
	return exported.Tendermint
}

type WasmConfig struct {
	InitConsensusState wasmtypes.ConsensusState
	InitClientState    wasmtypes.ClientState
	UpdateHeader       wasmtypes.Header
}

func (tmcfg *WasmConfig) GetClientType() string {
	return exported.Wasm
}

func NewWasmConfig(consensusState wasmtypes.ConsensusState, clientState wasmtypes.ClientState, updateHeader wasmtypes.Header) *WasmConfig {
	return &WasmConfig{
		InitConsensusState: consensusState,
		InitClientState:    clientState,
		UpdateHeader:       updateHeader,
	}
}

type ConnectionConfig struct {
	DelayPeriod uint64
	Version     *connectiontypes.Version
}

func NewConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		DelayPeriod: DefaultDelayPeriod,
		Version:     ConnectionVersion,
	}
}

type ChannelConfig struct {
	PortID  string
	Version string
	Order   channeltypes.Order
}

func NewChannelConfig() *ChannelConfig {
	return &ChannelConfig{
		PortID:  mock.PortID,
		Version: DefaultChannelVersion,
		Order:   channeltypes.UNORDERED,
	}
}
