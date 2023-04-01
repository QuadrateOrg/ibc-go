package types_test

import (
	"encoding/base64"

	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (suite *WasmTestSuite) TestVerifyHeaderGrandpa() {
	var (
		clientMsg   exported.ClientMessage
		clientState exported.ClientState
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		/*{
			"successful verify header", func() {},
			true,
		},
		{
			"unsuccessful verify header: para id mismatch", func() {
				cs, err := base64.StdEncoding.DecodeString(suite.testData["client_state_para_id_mismatch"])
				suite.Require().NoError(err)

				clientState = &wasmtypes.ClientState{
					Data:   cs,
					CodeId: suite.codeID,
					LatestHeight: clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 36,
					},
				}
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, "08-wasm-0", clientState)
			},
			false,
		},
		{
			"unsuccessful verify header: head height < consensus height", func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header_old"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Header{
					Data: data,
					Height: clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 29,
					},
				}
			},
			false,
		},*/
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupWithChannel()
			clientState = suite.clientState
			data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
			suite.Require().NoError(err)
			clientMsg = &wasmtypes.Header{
				Data: data,
				Height: clienttypes.Height{
					RevisionNumber: 2000,
					RevisionHeight: 39,
				},
			}

			tc.setup()
			err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *WasmTestSuite) TestUpdateStateTendermint() {
	var (
		path               *ibctesting.Path
		clientMessage      exported.ClientMessage
		clientStore        sdk.KVStore
		//consensusHeights   []exported.Height
		//prevClientState    exported.ClientState
		//prevConsensusState exported.ConsensusState
	)

	testCases := []struct {
		name      string
		malleate  func()
		expResult func()
		expPass   bool
	}{
		/*{
			"success with height later than latest height", func() {
				wasmHeader, ok := clientMessage.(*wasmtypes.Header)
				suite.Require().True(ok)
				suite.Require().True(path.EndpointA.GetClientState().GetLatestHeight().LT(wasmHeader.Height))
			},
			func() {
				wasmHeader, ok := clientMessage.(*wasmtypes.Header)
				suite.Require().True(ok)

				clientState := path.EndpointA.GetClientState()
				suite.Require().True(clientState.GetLatestHeight().EQ(wasmHeader.Height)) // new update, updated client state should have changed
				suite.Require().True(clientState.GetLatestHeight().EQ(consensusHeights[0]))
			}, true,
		},*/
	}
	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmTendermint() // reset
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			// ensure counterparty state is committed
			suite.coordinator.CommitBlock(suite.chainB)
			clientMessage, err = path.EndpointA.Chain.ConstructUpdateWasmClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
			suite.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			if tc.expPass {
				//consensusHeights = clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, clientMessage)

				header := clientMessage.(*wasmtypes.Header)
				var eHeader exported.ClientMessage
				err := suite.chainA.Codec.UnmarshalInterface(header.Data, &eHeader)
				tmHeader := eHeader.(*ibctm.Header)
				suite.Require().NoError(err)
				expTmConsensusState := &ibctm.ConsensusState{
					Timestamp:          tmHeader.GetTime(),
					Root:               commitmenttypes.NewMerkleRoot(tmHeader.Header.GetAppHash()),
					NextValidatorsHash: tmHeader.Header.NextValidatorsHash,
				}
				wasmData, err := suite.chainA.Codec.MarshalInterface(expTmConsensusState)
				suite.Require().NoError(err)
				expWasmConsensusState := &wasmtypes.ConsensusState{
					Data: wasmData,
					Timestamp: expTmConsensusState.GetTimestamp(),
				}

				bz := clientStore.Get(host.ConsensusStateKey(header.Height))
				updatedConsensusState := clienttypes.MustUnmarshalConsensusState(suite.chainA.App.AppCodec(), bz)

				suite.Require().Equal(expWasmConsensusState, updatedConsensusState)

			} else {
				suite.Require().Panics(func() {
					clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, clientMessage)
				})
			}

			// perform custom checks
			tc.expResult()
		})
	}
}
func (suite *WasmTestSuite) TestUpdateStateGrandpa() {
	var (
		clientMsg   exported.ClientMessage
		clientState exported.ClientState
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		/*{
			"success with height later than latest height",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Header{
					Data: data,
					Height: clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 39,
					},
				}
				// VerifyClientMessage must be run first
				err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"success with not verifying client message",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["header"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Header{
					Data: data,
					Height: clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 39,
					},
				}
			},
			true,
		},
		{
			"invalid ClientMessage type", func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					Data: data,
				}
			},
			false,
		},*/
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWithChannel()
			clientState = suite.clientState
			tc.setup()

			if tc.expPass {
				consensusHeights := clientState.UpdateState(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

				clientStateBz := suite.store.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)

				suite.Require().Len(consensusHeights, 1)
				suite.Require().Equal(clienttypes.Height{
					RevisionNumber: 2000,
					RevisionHeight: 39,
				}, consensusHeights[0])
				suite.Require().Equal(consensusHeights[0], newClientState.(*wasmtypes.ClientState).LatestHeight)
			} else {
				suite.Require().Panics(func() {
					clientState.UpdateState(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				})
			}
		})
	}
}

func (suite *WasmTestSuite) TestUpdateStateOnMisbehaviourGrandpa() {
	var (
		clientMsg   exported.ClientMessage
		clientState exported.ClientState
	)

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		/*{
			"successful update",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					Data: data,
				}
				clientState = suite.clientState
			},
			true,
		},*/
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWithChannel()
			tc.setup()

			if tc.expPass {
				suite.Require().NotPanics(func() {
					clientState.UpdateStateOnMisbehaviour(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				})
				clientStateBz := suite.store.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)
				status := newClientState.Status(suite.ctx, suite.store, suite.chainA.Codec)
				suite.Require().Equal(exported.Frozen, status)
			} else {
				suite.Require().Panics(func() {
					clientState.UpdateStateOnMisbehaviour(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				})
			}
		})
	}
}

func (suite *WasmTestSuite) TestUpdateStateOnMisbehaviourTendermint() {
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		/*{
			"success",
			func() {},
			true,
		},*/
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmTendermint()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)
	
			misbehaviourHeader, err := path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
			suite.Require().NoError(err)
			tmMisbehaviour := &ibctm.Misbehaviour{
				Header1: misbehaviourHeader,
				Header2: misbehaviourHeader,
			}
			wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
			suite.Require().NoError(err)
			clientMessage := &wasmtypes.Misbehaviour{
				Data: wasmData,
			}
			clientState.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, clientMessage)

			if tc.expPass {
				clientStateBz := clientStore.Get(host.ClientStateKey())
				suite.Require().NotEmpty(clientStateBz)

				newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)
				newWasmClientState := newClientState.(*wasmtypes.ClientState)

				var innerClientState exported.ClientState
				err = suite.chainA.Codec.UnmarshalInterface(newWasmClientState.Data, &innerClientState)
				suite.Require().Equal(misbehaviourHeader.GetHeight(), innerClientState.(*ibctm.ClientState).FrozenHeight)

				status := clientState.Status(suite.chainA.GetContext(), clientStore, suite.chainA.Codec)
				suite.Require().Equal(exported.Frozen, status)
			}
		})
	}
}