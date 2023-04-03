package types_test

import (
	"encoding/base64"
	"time"

	tmtypes "github.com/cometbft/cometbft/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibctestingmock "github.com/cosmos/ibc-go/v7/testing/mock"
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

func (suite *WasmTestSuite) TestVerifyHeaderTendermint() {
	var (
		path   *ibctesting.Path
		header *wasmtypes.Header
	)

	// Setup different validators and signers for testing different types of updates
	altPrivVal := ibctestingmock.NewPV()
	altPubKey, err := altPrivVal.GetPubKey()
	suite.Require().NoError(err)

	revisionHeight := int64(height.RevisionHeight)

	// create modified heights to use for test-cases
	altVal := tmtypes.NewValidator(altPubKey, 100)
	// Create alternative validator set with only altVal, invalid update (too much change in valSet)
	altValSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{altVal})
	altSigners := getAltSigners(altVal, altPrivVal)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name:     "success",
			malleate: func() {},
			expPass:  true,
		},
		{
			name: "successful verify header for header with a previous height",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// passing the CurrentHeader.Height as the block height as it will become a previous height once we commit N blocks
				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				// commit some blocks so that the created Header now has a previous height as the BlockHeight
				suite.coordinator.CommitNBlocks(suite.chainB, 5)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
			},
			expPass: true,
		},
		{
			name: "successful verify header: header with future height and different validator set",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altVal))
				bothSigners := suite.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+5, trustedHeight, suite.chainB.CurrentHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners)
			},
			expPass: true,
		},
		{
			name: "successful verify header: header with next height and different validator set",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altVal))
				bothSigners := suite.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, bothValSet, suite.chainB.NextVals, trustedVals, bothSigners)
			},
			expPass: true,
		},
		{
			name: "unsuccessful updates, passed in incorrect trusted validators for given consensus state",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				// Create bothValSet with both suite validator and altVal
				bothValSet := tmtypes.NewValidatorSet(append(suite.chainB.Vals.Validators, altVal))
				bothSigners := suite.chainB.Signers
				bothSigners[altVal.Address.String()] = altPrivVal

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, bothValSet, bothValSet, bothValSet, bothSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header with next height: update header mismatches nextValSetHash",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// this will err as altValSet.Hash() != consState.NextValidatorsHash
				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, altValSet, trustedVals, altSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update with future height: too much change in validator set",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, altValSet, altValSet, trustedVals, altSigners)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header height revision and trusted height revision mismatch",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)
				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				header = suite.chainB.CreateWasmClientHeader(chainIDRevision1, 3, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header height < consensus height",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				heightMinus1 := clienttypes.NewHeight(trustedHeight.RevisionNumber, trustedHeight.RevisionHeight-1)

				// Make new header at height less than latest client state
				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, int64(heightMinus1.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header basic validation failed",
			malleate: func() {
				var wasmData exported.ClientMessage
				err := suite.chainA.Codec.UnmarshalInterface(header.Data, &wasmData)
				suite.Require().NoError(err)

				tmHeader, ok := wasmData.(*ibctm.Header)
				suite.Require().True(ok)

				// cause header to fail validatebasic by changing commit height to mismatch header height
				tmHeader.SignedHeader.Commit.Height = revisionHeight - 1

				header.Data, err = suite.chainA.Codec.MarshalInterface(tmHeader)
				suite.Require().NoError(err)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header timestamp is not past last client timestamp",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time.Add(-time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful verify header: header with incorrect header chain-id",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				header = suite.chainB.CreateWasmClientHeader(chainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		{
			name: "unsuccessful update: trusting period has passed since last client timestamp",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight))
				suite.Require().True(found)

				header = suite.chainA.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+1, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				suite.chainB.ExpireClient(ibctesting.TrustingPeriod)
			},
			expPass: false,
		},
		/*{
			name: "unsuccessful update for a previous revision",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// passing the CurrentHeader.Height as the block height as it will become an update to previous revision once we upgrade the client
				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				// increment the revision of the chain
				err = path.EndpointB.UpgradeChain()
				suite.Require().NoError(err)
			},
			expPass: false,
		},*/
		{
			name: "successful update with identical header to a previous update",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// passing the CurrentHeader.Height as the block height as it will become a previous height once we commit N blocks
				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				// update client so the header constructed becomes a duplicate
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
			},
			expPass: true,
		},
		{
			name: "unsuccessful update to a future revision",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID+"-1", suite.chainB.CurrentHeader.Height+5, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},
		/*{
			name: "unsuccessful update: header height revision and trusted height revision mismatch",
			malleate: func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				// increment the revision of the chain
				err = path.EndpointB.UpgradeChain()
				suite.Require().NoError(err)

				header = suite.chainB.CreateWasmClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
			},
			expPass: false,
		},*/
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
		suite.SetupWasmTendermint()
		path = ibctesting.NewPath(suite.chainA, suite.chainB)

		err := path.EndpointA.CreateClient()
		suite.Require().NoError(err)

		// ensure counterparty state is committed
		suite.coordinator.CommitBlock(suite.chainB)
		header, err = path.EndpointA.Chain.ConstructUpdateWasmClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
		suite.Require().NoError(err)

		tc.malleate()

		clientState := path.EndpointA.GetClientState()

		clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

		err = clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), clientStore, header)

		if tc.expPass {
			suite.Require().NoError(err, tc.name)
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