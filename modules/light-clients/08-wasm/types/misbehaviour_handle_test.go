package types_test

import (
	//"encoding/base64"
	"time"
	"fmt"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	//solomachine "github.com/cosmos/ibc-go/v7/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

func (suite *WasmTestSuite) TestVerifyMisbehaviourGrandpa() {
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
			"successful misbehaviour verification",
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
				clientState.UpdateState(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

				// Reset client state to the previous for the test
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, "08-wasm-0", clientState)

				data, err = base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					Data: data,
				}
			},
			true,
		},
		{
			"trusted consensus state does not exist",
			func() {
				data, err := base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					Data: data,
				}
			},
			false,
		},
		{
			"invalid wasm misbehaviour",
			func() {
				clientMsg = &solomachine.Misbehaviour{}
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

			err := clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *WasmTestSuite) TestCheckForMisbehaviourGrandpa() {
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
			"valid update no misbehaviour",
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

				err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				suite.Require().NoError(err)
			},
			false,
		},
		{
			"valid fork misbehaviour returns true",
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
				clientState.UpdateState(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

				// Reset client state to the previous for the test
				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, "08-wasm-0", clientState)

				data, err = base64.StdEncoding.DecodeString(suite.testData["misbehaviour"])
				suite.Require().NoError(err)
				clientMsg = &wasmtypes.Misbehaviour{
					Data: data,
				}

				err = clientState.VerifyClientMessage(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"invalid wasm misbehaviour",
			func() {
				clientMsg = &solomachine.Misbehaviour{}
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
			foundMisbehaviour := clientState.CheckForMisbehaviour(suite.ctx, suite.chainA.Codec, suite.store, clientMsg)

			if tc.expPass {
				suite.Require().True(foundMisbehaviour)
			} else {
				suite.Require().False(foundMisbehaviour)
			}
		})
	}
}

func (suite *WasmTestSuite) TestCheckForMisbehaviourTendermint() {
	var (
		path          *ibctesting.Path
		clientMessage exported.ClientMessage
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid update no misbehaviour",
			func() {},
			false,
		},
		{
			"consensus state already exists, already updated",
			func() {
				wasmHeader, ok := clientMessage.(*wasmtypes.Header)
				suite.Require().True(ok)

				var wasmData exported.ClientMessage
				err := suite.chainA.Codec.UnmarshalInterface(wasmHeader.Data, &wasmData)
				suite.Require().NoError(err)

				tmHeader, ok := wasmData.(*ibctm.Header)
				suite.Require().NoError(err)

				tmConsensusState := &ibctm.ConsensusState{
					Timestamp:          tmHeader.GetTime(),
					Root:               commitmenttypes.NewMerkleRoot(tmHeader.Header.GetAppHash()),
					NextValidatorsHash: tmHeader.Header.NextValidatorsHash,
				}

				wasmDataCS, err := suite.chainA.Codec.MarshalInterface(tmConsensusState)
				suite.Require().NoError(err)
				wasmConsensusState := &wasmtypes.ConsensusState{
					Data: wasmDataCS,
					Timestamp: tmConsensusState.GetTimestamp(),
				}

				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(
					suite.chainA.GetContext(), path.EndpointA.ClientID, tmHeader.GetHeight(), wasmConsensusState)
			},
			false,
		},
		{
			"invalid fork misbehaviour: identical headers", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				err := path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				height := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				misbehaviourHeader := suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, int64(height.RevisionHeight), trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
				tmMisbehaviour := &ibctm.Misbehaviour{
					Header1: misbehaviourHeader,
					Header2: misbehaviourHeader,
				}
				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
				suite.Require().NoError(err)
				clientMessage = &wasmtypes.Misbehaviour{
					Data: wasmData,
				}
			}, false,
		},
		{
			"invalid time misbehaviour: monotonically increasing time", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				header1 := suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)
				header2 := suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers)

				tmMisbehaviour := &ibctm.Misbehaviour{
					Header1: header1,
					Header2: header2,
				}
				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
				suite.Require().NoError(err)
				clientMessage = &wasmtypes.Misbehaviour{
					Data: wasmData,
				}
			}, false,
		},
		{
			"consensus state already exists, app hash mismatch",
			func() {
				wasmHeader, ok := clientMessage.(*wasmtypes.Header)
				suite.Require().True(ok)

				var wasmData exported.ClientMessage
				err := suite.chainA.Codec.UnmarshalInterface(wasmHeader.Data, &wasmData)
				suite.Require().NoError(err)

				tmHeader, ok := wasmData.(*ibctm.Header)
				suite.Require().NoError(err)

				tmConsensusState := &ibctm.ConsensusState{
					Timestamp:          tmHeader.GetTime(),
					Root:               commitmenttypes.NewMerkleRoot([]byte{}), // empty bytes
					NextValidatorsHash: tmHeader.Header.NextValidatorsHash,
				}

				wasmDataCS, err := suite.chainA.Codec.MarshalInterface(tmConsensusState)
				suite.Require().NoError(err)
				wasmConsensusState := &wasmtypes.ConsensusState{
					Data: wasmDataCS,
					Timestamp: tmConsensusState.GetTimestamp(),
				}

				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientConsensusState(
					suite.chainA.GetContext(), path.EndpointA.ClientID, tmHeader.GetHeight(), wasmConsensusState)
			},
			true,
		},
		{
			"previous consensus state exists and header time is before previous consensus state time",
			func() {
				wasmHeader, ok := clientMessage.(*wasmtypes.Header)
				suite.Require().True(ok)

				var wasmData exported.ClientMessage
				err := suite.chainA.Codec.UnmarshalInterface(wasmHeader.Data, &wasmData)
				suite.Require().NoError(err)

				tmHeader, ok := wasmData.(*ibctm.Header)
				suite.Require().NoError(err)
				fmt.Println("Height: ", tmHeader.GetHeight())
				fmt.Println("Trusted Height: ", tmHeader.TrustedHeight)

				// offset header timestamp before previous consensus state timestamp
				tmHeader.Header.Time = tmHeader.GetTime().Add(-time.Hour)

				wasmHeader.Data, err = suite.chainA.Codec.MarshalInterface(tmHeader)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"next consensus state exists and header time is after next consensus state time",
			func() {
				wasmHeader, ok := clientMessage.(*wasmtypes.Header)
				suite.Require().True(ok)

				var wasmData exported.ClientMessage
				err := suite.chainA.Codec.UnmarshalInterface(wasmHeader.Data, &wasmData)
				suite.Require().NoError(err)

				tmHeader, ok := wasmData.(*ibctm.Header)
				suite.Require().NoError(err)
				fmt.Println("Height: ", tmHeader.Header.GetHeight())
				fmt.Println("Trusted Height: ", tmHeader.TrustedHeight)

				// offset header timestamp before previous consensus state timestamp
				tmHeader.Header.Time = tmHeader.GetTime().Add(time.Hour)

				wasmHeader.Data, err = suite.chainA.Codec.MarshalInterface(tmHeader)
				suite.Require().NoError(err)
				// commit block and update client, adding a new consensus state
				suite.coordinator.CommitBlock(suite.chainB)
				
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"valid fork misbehaviour returns true",
			func() {
				//header1, err := path.EndpointB.Chain.ConstructUpdateTMClientHeader(path.EndpointB.Counterparty.Chain, path.EndpointB.ClientID)
				header1, err := path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
				suite.Require().NoError(err)

				// commit block and update client
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				//header2, err := path.EndpointB.Chain.ConstructUpdateTMClientHeader(path.EndpointB.Counterparty.Chain, path.EndpointB.ClientID)
				header2, err := path.EndpointA.Chain.ConstructUpdateTMClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
				suite.Require().NoError(err)

				// assign the same height, each header will have a different commit hash
				header1.Header.Height = header2.Header.Height
				header1.Commit.Height = header2.Commit.Height

				tmMisbehaviour := &ibctm.Misbehaviour{
					Header1:  header1,
					Header2:  header2,
					//ClientId: path.EndpointA.ClientID,
				}

				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
				suite.Require().NoError(err)
				clientMessage = &wasmtypes.Misbehaviour{
					Data: wasmData,
				}

			},
			true,
		},
		{
			"valid time misbehaviour: not monotonically increasing time", func() {
				trustedHeight := path.EndpointA.GetClientState().GetLatestHeight().(clienttypes.Height)

				trustedVals, found := suite.chainB.GetValsAtHeight(int64(trustedHeight.RevisionHeight) + 1)
				suite.Require().True(found)

				tmMisbehaviour := &ibctm.Misbehaviour{
					Header2: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height+3, trustedHeight, suite.chainB.CurrentHeader.Time.Add(time.Minute), suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
					Header1: suite.chainB.CreateTMClientHeader(suite.chainB.ChainID, suite.chainB.CurrentHeader.Height, trustedHeight, suite.chainB.CurrentHeader.Time, suite.chainB.Vals, suite.chainB.NextVals, trustedVals, suite.chainB.Signers),
				}

				wasmData, err := suite.chainB.Codec.MarshalInterface(tmMisbehaviour)
				suite.Require().NoError(err)
				clientMessage = &wasmtypes.Misbehaviour{
					Data: wasmData,
				}
			}, true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmTendermint()
			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			err := path.EndpointA.CreateClient()
			suite.Require().NoError(err)

			// ensure counterparty state is committed
			suite.coordinator.CommitBlock(suite.chainB)
			clientMessage, err = path.EndpointA.Chain.ConstructUpdateWasmClientHeader(path.EndpointA.Counterparty.Chain, path.EndpointA.ClientID)
			suite.Require().NoError(err)

			tc.malleate()

			clientState := path.EndpointA.GetClientState()
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			foundMisbehaviour := clientState.CheckForMisbehaviour(
				suite.chainA.GetContext(),
				suite.chainA.App.AppCodec(),
				clientStore, // pass in clientID prefixed clientStore
				clientMessage,
			)

			if tc.expPass {
				suite.Require().True(foundMisbehaviour)
			} else {
				suite.Require().False(foundMisbehaviour)
			}
		})
	}
}