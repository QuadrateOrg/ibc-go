package wasm_test

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v5/modules/core/exported"
	wasm "github.com/cosmos/ibc-go/v5/modules/light-clients/10-wasm/types"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
	"github.com/cosmos/ibc-go/v5/testing/simapp"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

type WasmTestSuite struct {
	suite.Suite
	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	ctx         sdk.Context
	cdc         codec.Codec
	now         time.Time
	store       sdk.KVStore
}

func (suite *WasmTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)

	// TODO: deprecate usage in favor of testing package
	checkTx := false
	app := simapp.Setup(checkTx)
	suite.cdc = app.AppCodec()
	suite.now = time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	suite.ctx = app.BaseApp.NewContext(checkTx, tmproto.Header{Height: 1, Time: suite.now})

	wasmConfig := wasm.VMConfig{
		DataDir:           "tmp",
		SupportedFeatures: []string{"storage", "iterator"},
		MemoryLimitMb:     uint32(math.Pow(2, 12)),
		PrintDebug:        true,
		CacheSizeMb:       uint32(math.Pow(2, 8)),
	}
	validationConfig := wasm.ValidationConfig{
		MaxSizeAllowed: int(math.Pow(2, 26)),
	}
	suite.store = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), exported.Wasm)
	clientState := wasm.ClientState{}
	os.MkdirAll("tmp", 0o755)
	wasm.CreateVM(&wasmConfig, &validationConfig)
	data, err := os.ReadFile("ics10_grandpa_cw.wasm")

	suite.Require().NoError(err)
	err = wasm.PushNewWasmCode(suite.store, &clientState, data)
	suite.Require().NoError(err)
	consensusState := wasm.ConsensusState{
		CodeId: clientState.CodeId,
	}
	err = clientState.Initialize(suite.ctx, suite.cdc, suite.store, &consensusState)
	suite.Require().NoError(err)

	value := suite.store.Get([]byte("client_test_item"))
	suite.Require().NotNil(value)
	fmt.Println("Value: ", string(value[:]))
}

func (suite *WasmTestSuite) TestWasm() {
	suite.Run("Init contract", func() {})
}

func TestWasmTestSuite(t *testing.T) {
	suite.Run(t, new(WasmTestSuite))
}
