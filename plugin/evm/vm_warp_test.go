// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"math/big"
	"sort"
	"testing"
	"time"

	_ "embed"

	commonEng "github.com/luxfi/consensus/core"
	consensuscontext "github.com/luxfi/consensus/context"
	"github.com/luxfi/consensus/engine/chain/block"
	"github.com/luxfi/consensus/validator"
	"github.com/luxfi/consensus/validator/validatorstest"
	"github.com/luxfi/crypto"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/crypto/bls/signer/localsigner"
	"github.com/luxfi/evm/consensus"
	"github.com/luxfi/evm/core"
	"github.com/luxfi/evm/eth/tracers"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/params/extras"
	customheader "github.com/luxfi/evm/plugin/evm/header"
	"github.com/luxfi/evm/precompile/contract"
	warpcontract "github.com/luxfi/evm/precompile/contracts/warp"
	"github.com/luxfi/evm/predicate"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/evm/utils/utilstest"
	"github.com/luxfi/evm/warp"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	luxfiDatabase "github.com/luxfi/database"
	"github.com/luxfi/database/memdb"
	"github.com/luxfi/database/prefixdb"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/node/network/p2p"
	"github.com/luxfi/node/network/p2p/lp118"
	"github.com/luxfi/node/proto/pb/sdk"
	luxdUtils "github.com/luxfi/node/utils"
	"github.com/luxfi/node/utils/constants"
	luxWarp "github.com/luxfi/warp"
	"github.com/luxfi/warp/payload"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var (
	//go:embed ExampleWarp.bin
	exampleWarpBin string
	//go:embed ExampleWarp.abi
	exampleWarpABI string
)

type warpMsgFrom int

const (
	fromSubnet warpMsgFrom = iota
	fromPrimary
)

type useWarpMsgSigners int

const (
	signersSubnet useWarpMsgSigners = iota
	signersPrimary
)

func TestSendWarpMessage(t *testing.T) {
	for _, scheme := range schemes {
		t.Run(scheme, func(t *testing.T) {
			testSendWarpMessage(t, scheme)
		})
	}
}

func testSendWarpMessage(t *testing.T, scheme string) {
	require := require.New(t)
	genesis := &core.Genesis{}

	require.NoError(genesis.UnmarshalJSON([]byte(toGenesisJSON(forkToChainConfig["Latest"]))))
	// Copy network upgrades from source config and add warp precompile
	sourceExtra := params.GetExtra(forkToChainConfig["Latest"])
	extra := params.GetExtra(genesis.Config)
	extra.NetworkUpgrades = sourceExtra.NetworkUpgrades
	extra.GenesisPrecompiles = extras.Precompiles{
		warpcontract.ConfigKey: warpcontract.NewDefaultConfig(utils.NewUint64(0)), // Enable at genesis
	}
	genesisJSON := marshalGenesisWithExtras(genesis)
	tvm := newVM(t, testVMConfig{
		genesisJSON: genesisJSON,
		configJSON:  getConfig(scheme, ""),
	})

	defer func() {
		require.NoError(tvm.vm.Shutdown(context.Background()))
	}()

	acceptedLogsChan := make(chan []*types.Log, 10)
	logsSub := tvm.vm.eth.APIBackend.SubscribeAcceptedLogsEvent(acceptedLogsChan)
	defer logsSub.Unsubscribe()

	payloadData := luxdUtils.RandomBytes(100)

	warpSendMessageInput, err := warpcontract.PackSendWarpMessage(payloadData)
	require.NoError(err)
	addressedPayload, err := payload.NewAddressedCall(
		testEthAddrs[0].Bytes(),
		payloadData,
	)
	require.NoError(err)
	// Use consensuscontext to match what the warp precompile uses
	chainID := consensuscontext.GetChainID(tvm.vm.ctx)
	expectedUnsignedMessage, err := luxWarp.NewUnsignedMessage(
		consensuscontext.GetNetworkID(tvm.vm.ctx),
		chainID[:],
		addressedPayload.Bytes(),
	)
	require.NoError(err)

	// Submit a transaction to trigger sending a warp message
	tx0 := types.NewTransaction(uint64(0), warpcontract.ContractAddress, big.NewInt(1), 100_000, big.NewInt(testMinGasPrice), warpSendMessageInput)
	key0ECDSA, err := crypto.ToECDSA(testKeys[0].Bytes())
	require.NoError(err)
	signedTx0, err := types.SignTx(tx0, types.LatestSignerForChainID(tvm.vm.chainConfig.ChainID), key0ECDSA)
	require.NoError(err)

	errs := tvm.vm.txPool.AddRemotesSync([]*types.Transaction{signedTx0})
	require.NoError(errs[0])

	msg, err := tvm.vm.WaitForEvent(context.Background())
	require.NoError(err)
	require.Equal(commonEng.PendingTxs, msg)

	blk, err := tvm.vm.BuildBlock(context.Background())
	require.NoError(err)

	require.NoError(blk.Verify(context.Background()))

	// Verify that the constructed block contains the expected log with an unsigned warp message in the log data
	// Get internal block safely using unwrapBlock helper
	internalBlock := unwrapBlock(t, blk)
	ethBlock1 := internalBlock.ethBlock
	require.Len(ethBlock1.Transactions(), 1)
	receipts := rawdb.ReadReceipts(tvm.vm.chaindb, ethBlock1.Hash(), ethBlock1.NumberU64(), ethBlock1.Time(), tvm.vm.chainConfig)
	require.Len(receipts, 1)

	require.Len(receipts[0].Logs, 1)
	expectedTopics := []common.Hash{
		warpcontract.WarpABI.Events["SendWarpMessage"].ID,
		common.BytesToHash(testEthAddrs[0].Bytes()),
		common.Hash(expectedUnsignedMessage.ID()),
	}
	require.Equal(expectedTopics, receipts[0].Logs[0].Topics)
	logData := receipts[0].Logs[0].Data
	unsignedMessage, err := warpcontract.UnpackSendWarpEventDataToMessage(logData)
	require.NoError(err)

	// Verify the signature cannot be fetched before the block is accepted
	_, err = tvm.vm.warpBackend.GetMessageSignature(context.Background(), unsignedMessage)
	require.Error(err)

	require.NoError(tvm.vm.SetPreference(context.Background(), blk.ID()))
	require.NoError(blk.Accept(context.Background()))
	tvm.vm.blockChain.DrainAcceptorQueue()

	// Verify the message signature after accepting the block.
	rawSignatureBytes, err := tvm.vm.warpBackend.GetMessageSignature(context.Background(), unsignedMessage)
	require.NoError(err)
	blsSignature, err := bls.SignatureFromBytes(rawSignatureBytes)
	require.NoError(err)

	select {
	case acceptedLogs := <-acceptedLogsChan:
		require.Len(acceptedLogs, 1, "unexpected length of accepted logs")
		require.Equal(acceptedLogs[0], receipts[0].Logs[0])
	case <-time.After(time.Second):
		require.Fail("Failed to read accepted logs from subscription")
	}

	// Verify the produced message signature is valid
	// Use the test warp holder to access the underlying BLS signer for verification
	testHolder := getTestWarpHolder()
	blsSigner := testHolder.GetBLSSigner()
	publicKey := blsSigner.PublicKey()
	require.True(bls.Verify(publicKey, blsSignature, unsignedMessage.Bytes()))

	// Verify the blockID will now be signed by the backend and produces a valid signature.
	blockID := blk.ID()
	rawBlockSignatureBytes, err := tvm.vm.warpBackend.GetBlockSignature(context.Background(), blockID)
	require.NoError(err)
	blsSignature, err = bls.SignatureFromBytes(rawBlockSignatureBytes)
	require.NoError(err)

	blockHashPayload, err := payload.NewHash(blockID)
	require.NoError(err)
	chainID = consensuscontext.GetChainID(tvm.vm.ctx)
	blockUnsignedMessage, err := luxWarp.NewUnsignedMessage(consensuscontext.GetNetworkID(tvm.vm.ctx), chainID[:], blockHashPayload.Bytes())
	require.NoError(err)

	// Verify the produced message signature is valid
	publicKey = blsSigner.PublicKey()
	require.True(bls.Verify(publicKey, blsSignature, blockUnsignedMessage.Bytes()))
}

func TestValidateWarpMessage(t *testing.T) {
	for _, scheme := range schemes {
		t.Run(scheme, func(t *testing.T) {
			testValidateWarpMessage(t, scheme)
		})
	}
}

func testValidateWarpMessage(t *testing.T, scheme string) {
	require := require.New(t)
	sourceChainID := ids.GenerateTestID()
	sourceAddress := common.HexToAddress("0x376c47978271565f56DEB45495afa69E59c16Ab2")
	payloadData := []byte{1, 2, 3}
	addressedPayload, err := payload.NewAddressedCall(
		sourceAddress.Bytes(),
		payloadData,
	)
	require.NoError(err)
	unsignedMessage, err := luxWarp.NewUnsignedMessage(testNetworkID, sourceChainID[:], addressedPayload.Bytes())
	require.NoError(err)

	exampleWarpABI := contract.ParseABI(exampleWarpABI)
	exampleWarpPayload, err := exampleWarpABI.Pack(
		"validateWarpMessage",
		uint32(0),
		sourceChainID,
		sourceAddress,
		payloadData,
	)
	require.NoError(err)

	testWarpVMTransaction(t, scheme, unsignedMessage, true, exampleWarpPayload)
}

func TestValidateInvalidWarpMessage(t *testing.T) {
	for _, scheme := range schemes {
		t.Run(scheme, func(t *testing.T) {
			testValidateInvalidWarpMessage(t, scheme)
		})
	}
}

func testValidateInvalidWarpMessage(t *testing.T, scheme string) {
	require := require.New(t)
	sourceChainID := ids.GenerateTestID()
	sourceAddress := common.HexToAddress("0x376c47978271565f56DEB45495afa69E59c16Ab2")
	payloadData := []byte{1, 2, 3}
	addressedPayload, err := payload.NewAddressedCall(
		sourceAddress.Bytes(),
		payloadData,
	)
	require.NoError(err)
	unsignedMessage, err := luxWarp.NewUnsignedMessage(testNetworkID, sourceChainID[:], addressedPayload.Bytes())
	require.NoError(err)

	exampleWarpABI := contract.ParseABI(exampleWarpABI)
	exampleWarpPayload, err := exampleWarpABI.Pack(
		"validateInvalidWarpMessage",
		uint32(0),
	)
	require.NoError(err)

	testWarpVMTransaction(t, scheme, unsignedMessage, false, exampleWarpPayload)
}

func TestValidateWarpBlockHash(t *testing.T) {
	for _, scheme := range schemes {
		t.Run(scheme, func(t *testing.T) {
			testValidateWarpBlockHash(t, scheme)
		})
	}
}

func testValidateWarpBlockHash(t *testing.T, scheme string) {
	require := require.New(t)
	sourceChainID := ids.GenerateTestID()
	blockHash := ids.GenerateTestID()
	blockHashPayload, err := payload.NewHash(blockHash)
	require.NoError(err)
	unsignedMessage, err := luxWarp.NewUnsignedMessage(testNetworkID, sourceChainID[:], blockHashPayload.Bytes())
	require.NoError(err)

	exampleWarpABI := contract.ParseABI(exampleWarpABI)
	exampleWarpPayload, err := exampleWarpABI.Pack(
		"validateWarpBlockHash",
		uint32(0),
		sourceChainID,
		blockHash,
	)
	require.NoError(err)

	testWarpVMTransaction(t, scheme, unsignedMessage, true, exampleWarpPayload)
}

func TestValidateInvalidWarpBlockHash(t *testing.T) {
	for _, scheme := range schemes {
		t.Run(scheme, func(t *testing.T) {
			testValidateInvalidWarpBlockHash(t, scheme)
		})
	}
}

func testValidateInvalidWarpBlockHash(t *testing.T, scheme string) {
	require := require.New(t)
	sourceChainID := ids.GenerateTestID()
	blockHash := ids.GenerateTestID()
	blockHashPayload, err := payload.NewHash(blockHash)
	require.NoError(err)
	unsignedMessage, err := luxWarp.NewUnsignedMessage(testNetworkID, sourceChainID[:], blockHashPayload.Bytes())
	require.NoError(err)

	exampleWarpABI := contract.ParseABI(exampleWarpABI)
	exampleWarpPayload, err := exampleWarpABI.Pack(
		"validateInvalidWarpBlockHash",
		uint32(0),
	)
	require.NoError(err)

	testWarpVMTransaction(t, scheme, unsignedMessage, false, exampleWarpPayload)
}

func testWarpVMTransaction(t *testing.T, scheme string, unsignedMessage *luxWarp.UnsignedMessage, validSignature bool, txPayload []byte) {
	require := require.New(t)
	genesis := &core.Genesis{}
	require.NoError(genesis.UnmarshalJSON([]byte(toGenesisJSON(forkToChainConfig["Latest"]))))
	// Copy network upgrades from source config and add warp precompile
	sourceExtra := params.GetExtra(forkToChainConfig["Latest"])
	extra := params.GetExtra(genesis.Config)
	extra.NetworkUpgrades = sourceExtra.NetworkUpgrades
	extra.GenesisPrecompiles = extras.Precompiles{
		warpcontract.ConfigKey: warpcontract.NewDefaultConfig(utils.NewUint64(0)), // Enable at genesis
	}
	genesisJSON := marshalGenesisWithExtras(genesis)
	tvm := newVM(t, testVMConfig{
		genesisJSON: genesisJSON,
		configJSON:  getConfig(scheme, ""),
	})

	defer func() {
		require.NoError(tvm.vm.Shutdown(context.Background()))
	}()

	acceptedLogsChan := make(chan []*types.Log, 10)
	logsSub := tvm.vm.eth.APIBackend.SubscribeAcceptedLogsEvent(acceptedLogsChan)
	defer logsSub.Unsubscribe()

	nodeID1 := ids.GenerateTestNodeID()
	blsSecretKey1, err := localsigner.New()
	require.NoError(err)
	blsPublicKey1 := blsSecretKey1.PublicKey()
	blsSignature1, err := blsSecretKey1.Sign(unsignedMessage.Bytes())
	require.NoError(err)

	nodeID2 := ids.GenerateTestNodeID()
	blsSecretKey2, err := localsigner.New()
	require.NoError(err)
	blsPublicKey2 := blsSecretKey2.PublicKey()
	blsSignature2, err := blsSecretKey2.Sign(unsignedMessage.Bytes())
	require.NoError(err)

	blsAggregatedSignature, err := bls.AggregateSignatures([]*bls.Signature{blsSignature1, blsSignature2})
	require.NoError(err)

	minimumValidPChainHeight := uint64(10)
	getValidatorSetTestErr := errors.New("can't get validator set test error")

	testValidatorState := &validatorstest.State{
		// GetSubnetIDF removed - not available in current interface
		GetValidatorSetF: func(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
			if height < minimumValidPChainHeight {
				return nil, getValidatorSetTestErr
			}
			return map[ids.NodeID]*validators.GetValidatorOutput{
				nodeID1: {
					NodeID:    nodeID1,
					PublicKey: bls.PublicKeyToCompressedBytes(blsPublicKey1),
					Weight:    50,
				},
				nodeID2: {
					NodeID:    nodeID2,
					PublicKey: bls.PublicKeyToCompressedBytes(blsPublicKey2),
					Weight:    50,
				},
			}, nil
		},
	}
	wrappedValidatorState := utilstest.NewTestValidatorStateFromBase(testValidatorState)
	tvm.vm.ctx = consensuscontext.WithValidatorState(tvm.vm.ctx, wrappedValidatorState)

	signersBitSet := set.NewBits()
	signersBitSet.Add(0)
	signersBitSet.Add(1)

	warpSignature := &luxWarp.BitSetSignature{
		Signers: signersBitSet.Bytes(),
	}

	blsAggregatedSignatureBytes := bls.SignatureToBytes(blsAggregatedSignature)
	copy(warpSignature.Signature[:], blsAggregatedSignatureBytes)

	signedMessage, err := luxWarp.NewMessage(
		unsignedMessage,
		warpSignature,
	)
	require.NoError(err)

	createTx, err := types.SignTx(
		types.NewContractCreation(0, common.Big0, 7_000_000, big.NewInt(225*utils.GWei), common.Hex2Bytes(exampleWarpBin)),
		types.LatestSignerForChainID(tvm.vm.chainConfig.ChainID),
		func() *ecdsa.PrivateKey {
			key, err := crypto.ToECDSA(testKeys[0].Bytes())
			require.NoError(err)
			return key
		}(),
	)
	require.NoError(err)
	cryptoAddr := crypto.Address(testEthAddrs[0])
	exampleWarpAddress := common.Address(crypto.CreateAddress(cryptoAddr, 0))

	tx, err := types.SignTx(
		predicate.NewPredicateTx(
			tvm.vm.chainConfig.ChainID,
			1,
			&exampleWarpAddress,
			1_000_000,
			big.NewInt(225*utils.GWei),
			big.NewInt(200*utils.GWei), // Increased from 1 GWei to cover block gas cost
			common.Big0,
			txPayload,
			types.AccessList{},
			warpcontract.ContractAddress,
			signedMessage.Bytes(),
		),
		types.LatestSignerForChainID(tvm.vm.chainConfig.ChainID),
		func() *ecdsa.PrivateKey {
			key, err := crypto.ToECDSA(testKeys[0].Bytes())
			require.NoError(err)
			return key
		}(),
	)
	require.NoError(err)
	errs := tvm.vm.txPool.AddRemotesSync([]*types.Transaction{createTx, tx})
	for i, err := range errs {
		require.NoError(err, "failed to add tx at index %d", i)
	}

	// If [validSignature] set the signature to be considered valid at the verified height.
	blockCtx := &block.Context{
		PChainHeight: minimumValidPChainHeight - 1,
	}
	if validSignature {
		blockCtx.PChainHeight = minimumValidPChainHeight
	}
	tvm.vm.clock.Set(tvm.vm.clock.Time().Add(2 * time.Second))

	msg, err := tvm.vm.WaitForEvent(context.Background())
	require.NoError(err)
	require.Equal(commonEng.PendingTxs, msg)

	warpBlock, err := tvm.vm.BuildBlockWithContext(context.Background(), blockCtx)
	require.NoError(err)

	warpBlockVerifyWithCtx, ok := warpBlock.(block.WithVerifyContext)
	require.True(ok)
	shouldVerifyWithCtx, err := warpBlockVerifyWithCtx.ShouldVerifyWithContext(context.Background())
	require.NoError(err)
	require.True(shouldVerifyWithCtx)
	require.NoError(warpBlockVerifyWithCtx.VerifyWithContext(context.Background(), blockCtx))
	require.NoError(tvm.vm.SetPreference(context.Background(), warpBlock.ID()))
	require.NoError(warpBlock.Accept(context.Background()))
	tvm.vm.blockChain.DrainAcceptorQueue()

	// Get internal block safely using unwrapBlock helper
	warpInternalBlock := unwrapBlock(t, warpBlock)
	ethBlock := warpInternalBlock.ethBlock
	verifiedMessageReceipts := tvm.vm.blockChain.GetReceiptsByHash(ethBlock.Hash())
	require.Len(verifiedMessageReceipts, 2)
	for i, receipt := range verifiedMessageReceipts {
		require.Equal(types.ReceiptStatusSuccessful, receipt.Status, "index: %d", i)
	}

	tracerAPI := tracers.NewAPI(tvm.vm.eth.APIBackend)
	txTraceResults, err := tracerAPI.TraceBlockByHash(context.Background(), ethBlock.Hash(), nil)
	require.NoError(err)
	require.Len(txTraceResults, 2)
	blockTxTraceResultBytes, err := json.Marshal(txTraceResults[1].Result)
	require.NoError(err)
	unmarshalResults := make(map[string]interface{})
	require.NoError(json.Unmarshal(blockTxTraceResultBytes, &unmarshalResults))
	// The tracer returns "0x" for empty return data (hex encoding of empty bytes)
	require.Equal("0x", unmarshalResults["returnValue"])

	txTraceResult, err := tracerAPI.TraceTransaction(context.Background(), tx.Hash(), nil)
	require.NoError(err)
	txTraceResultBytes, err := json.Marshal(txTraceResult)
	require.NoError(err)
	require.JSONEq(string(txTraceResultBytes), string(blockTxTraceResultBytes))
}

func TestReceiveWarpMessage(t *testing.T) {
	for _, scheme := range schemes {
		t.Run(scheme, func(t *testing.T) {
			testReceiveWarpMessageWithScheme(t, scheme)
		})
	}
}

func testReceiveWarpMessageWithScheme(t *testing.T, scheme string) {
	require := require.New(t)
	fork := "Latest"
	tvm := newVM(t, testVMConfig{
		fork:       &fork,
		configJSON: getConfig(scheme, ""),
	})
	defer func() {
		require.NoError(tvm.vm.Shutdown(context.Background()))
	}()

	// enable warp at the default genesis time
	enableTime := time.Unix(0, 0)
	enableConfig := warpcontract.NewDefaultConfig(utils.TimeToNewUint64(enableTime))

	// disable warp so we can re-enable it with RequirePrimaryNetworkSigners
	disableTime := time.Unix(0, 0).Add(10 * time.Second)
	disableConfig := warpcontract.NewDisableConfig(utils.TimeToNewUint64(disableTime))

	// re-enable warp with RequirePrimaryNetworkSigners
	reEnableTime := disableTime.Add(10 * time.Second)
	reEnableConfig := warpcontract.NewConfig(
		utils.TimeToNewUint64(reEnableTime),
		0,    // QuorumNumerator
		true, // RequirePrimaryNetworkSigners
	)

	tvm.vm.chainConfigExtra().UpgradeConfig = extras.UpgradeConfig{
		PrecompileUpgrades: []extras.PrecompileUpgrade{
			{Config: enableConfig},
			{Config: disableConfig},
			{Config: reEnableConfig},
		},
	}

	type test struct {
		name          string
		sourceChainID ids.ID
		msgFrom       warpMsgFrom
		useSigners    useWarpMsgSigners
		blockTime     time.Time
	}

	blockGap := 2 * time.Second // Build blocks with a gap. Blocks built too quickly will have high fees.
	tests := []test{
		{
			name:          "subnet message should be signed by subnet without RequirePrimaryNetworkSigners",
			sourceChainID: consensuscontext.GetChainID(tvm.vm.ctx),
			msgFrom:       fromSubnet,
			useSigners:    signersSubnet,
			blockTime:     time.Unix(0, 0), // Genesis time
		},
		{
			name:          "P-Chain message should be signed by subnet without RequirePrimaryNetworkSigners",
			sourceChainID: constants.PlatformChainID,
			msgFrom:       fromPrimary,
			useSigners:    signersSubnet,
			blockTime:     time.Unix(0, 0).Add(blockGap), // Genesis + blockGap
		},
		{
			name:          "C-Chain message should be signed by subnet without RequirePrimaryNetworkSigners",
			sourceChainID: consensuscontext.GetChainID(tvm.vm.ctx),
			msgFrom:       fromPrimary,
			useSigners:    signersSubnet,
			blockTime:     time.Unix(0, 0).Add(2 * blockGap), // Genesis + 2*blockGap
		},
		// Note here we disable warp and re-enable it with RequirePrimaryNetworkSigners
		// by using reEnableTime.
		{
			name:          "subnet message should be signed by subnet with RequirePrimaryNetworkSigners (unimpacted)",
			sourceChainID: consensuscontext.GetChainID(tvm.vm.ctx),
			msgFrom:       fromSubnet,
			useSigners:    signersSubnet,
			blockTime:     reEnableTime,
		},
		{
			name:          "P-Chain message should be signed by subnet with RequirePrimaryNetworkSigners (unimpacted)",
			sourceChainID: constants.PlatformChainID,
			msgFrom:       fromPrimary,
			useSigners:    signersSubnet,
			blockTime:     reEnableTime.Add(blockGap),
		},
		{
			name:          "C-Chain message should be signed by primary with RequirePrimaryNetworkSigners (impacted)",
			sourceChainID: consensuscontext.GetChainID(tvm.vm.ctx),
			msgFrom:       fromPrimary,
			useSigners:    signersPrimary,
			blockTime:     reEnableTime.Add(2 * blockGap),
		},
	}
	// Note each test corresponds to a block, the tests must be ordered by block
	// time and cannot, eg be run in parallel or a separate golang test.
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testReceiveWarpMessage(
				t, tvm.vm, test.sourceChainID, test.msgFrom, test.useSigners, test.blockTime,
			)
		})
	}
}

func testReceiveWarpMessage(
	t *testing.T, vm *VM,
	sourceChainID ids.ID,
	msgFrom warpMsgFrom, useSigners useWarpMsgSigners,
	blockTime time.Time,
) {
	require := require.New(t)
	payloadData := luxdUtils.RandomBytes(100)
	addressedPayload, err := payload.NewAddressedCall(
		testEthAddrs[0].Bytes(),
		payloadData,
	)
	require.NoError(err)

	subnetID := ids.GenerateTestID()
	vm.ctx = consensus.WithSubnetID(vm.ctx, subnetID)
	vm.ctx = consensus.WithNetworkID(vm.ctx, testNetworkID)
	// Set the chain-to-subnet mapping so GetSubnetID returns the correct subnet
	// For subnet sources, map to the generated subnet
	// For primary network sources (except P-Chain), map to Empty (primary network)
	// P-Chain doesn't need a mapping as it's handled specially
	if msgFrom == fromSubnet {
		utilstest.SetChainSubnetMapping(sourceChainID, subnetID)
	} else if sourceChainID != constants.PlatformChainID {
		// Primary network source (e.g., "C-Chain" in test) - map to Empty
		utilstest.SetChainSubnetMapping(sourceChainID, ids.Empty)
	}
	defer utilstest.ClearChainSubnetMapping()
	unsignedMessage, err := luxWarp.NewUnsignedMessage(
		consensuscontext.GetNetworkID(vm.ctx),
		sourceChainID[:],
		addressedPayload.Bytes(),
	)
	require.NoError(err)

	type signer struct {
		nodeID    ids.NodeID
		secret    bls.Signer
		signature *bls.Signature
		weight    uint64
	}
	newSigner := func(weight uint64) signer {
		secret, err := localsigner.New()
		require.NoError(err)
		sig, err := secret.Sign(unsignedMessage.Bytes())
		require.NoError(err)

		return signer{
			nodeID:    ids.GenerateTestNodeID(),
			secret:    secret,
			signature: sig,
			weight:    weight,
		}
	}

	primarySigners := []signer{
		newSigner(50),
		newSigner(50),
	}
	subnetSigners := []signer{
		newSigner(50),
		newSigner(50),
	}
	signers := subnetSigners
	if useSigners == signersPrimary {
		signers = primarySigners
	}

	// Sort signers by public key for canonical ordering - this must match
	// the order used during verification (CanonicalValidatorSet sorts by pubkey)
	sort.Slice(signers, func(i, j int) bool {
		return bytes.Compare(
			bls.PublicKeyToCompressedBytes(signers[i].secret.PublicKey()),
			bls.PublicKeyToCompressedBytes(signers[j].secret.PublicKey()),
		) < 0
	})

	// Debug: print sorted public keys
	for i, s := range signers {
		pkBytes := bls.PublicKeyToCompressedBytes(s.secret.PublicKey())
		t.Logf("DEBUG TEST: signer[%d] pubkey=%x", i, pkBytes)
	}

	blsSignatures := make([]*bls.Signature, len(signers))
	for i := range signers {
		blsSignatures[i] = signers[i].signature
	}
	blsAggregatedSignature, err := bls.AggregateSignatures(blsSignatures)
	require.NoError(err)

	minimumValidPChainHeight := uint64(10)
	getValidatorSetTestErr := errors.New("can't get validator set test error")

	testValidatorState2 := &validatorstest.State{
		// GetSubnetIDF removed - not available in current interface
		GetValidatorSetF: func(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
			if height < minimumValidPChainHeight {
				return nil, getValidatorSetTestErr
			}
			signers := subnetSigners
			if subnetID == constants.PrimaryNetworkID {
				signers = primarySigners
			}

			vdrOutput := make(map[ids.NodeID]*validators.GetValidatorOutput)
			for _, s := range signers {
				vdrOutput[s.nodeID] = &validators.GetValidatorOutput{
					NodeID:    s.nodeID,
					PublicKey: bls.PublicKeyToCompressedBytes(s.secret.PublicKey()),
					Weight:    s.weight,
				}
			}
			return vdrOutput, nil
		},
	}
	wrappedValidatorState2 := utilstest.NewTestValidatorStateFromBase(testValidatorState2)
	vm.ctx = consensus.WithValidatorState(vm.ctx, wrappedValidatorState2)
	// Propagate the updated context to the blockchain so the miner can access the validator state
	vm.blockChain.SetConsensusContext(vm.ctx)

	signersBitSet := set.NewBits()
	for i := range signers {
		signersBitSet.Add(i)
	}

	warpSignature := &luxWarp.BitSetSignature{
		Signers: signersBitSet.Bytes(),
	}

	blsAggregatedSignatureBytes := bls.SignatureToBytes(blsAggregatedSignature)
	copy(warpSignature.Signature[:], blsAggregatedSignatureBytes)

	signedMessage, err := luxWarp.NewMessage(
		unsignedMessage,
		warpSignature,
	)
	require.NoError(err)

	getWarpMsgInput, err := warpcontract.PackGetVerifiedWarpMessage(0)
	require.NoError(err)
	// Use a higher gasTipCap (200 GWei) to ensure the transaction pays enough
	// block fee to cover the requiredBlockGasCost. The block fee verification
	// checks: (tip * gasUsed) / baseFee >= requiredBlockGasCost
	pendingNonce := vm.txPool.Nonce(testEthAddrs[0])
	t.Logf("DEBUG TEST: pendingNonce=%d, blockTime=%v", pendingNonce, blockTime)
	getVerifiedWarpMessageTx, err := types.SignTx(
		predicate.NewPredicateTx(
			vm.chainConfig.ChainID,
			pendingNonce,
			&warpcontract.Module.Address,
			1_000_000,
			big.NewInt(225*utils.GWei),
			big.NewInt(200*utils.GWei), // Increased from 1 GWei to cover block gas cost
			common.Big0,
			getWarpMsgInput,
			types.AccessList{},
			warpcontract.ContractAddress,
			signedMessage.Bytes(),
		),
		types.LatestSignerForChainID(vm.chainConfig.ChainID),
		func() *ecdsa.PrivateKey {
			key, err := crypto.ToECDSA(testKeys[0].Bytes())
			require.NoError(err)
			return key
		}(),
	)
	require.NoError(err)
	errs := vm.txPool.AddRemotesSync([]*types.Transaction{getVerifiedWarpMessageTx})
	for i, err := range errs {
		require.NoError(err, "failed to add tx at index %d", i)
	}

	// Build, verify, and accept block with valid proposer context.
	validProposerCtx := &block.Context{
		PChainHeight: minimumValidPChainHeight,
	}
	vm.clock.Set(blockTime)

	msg, err := vm.WaitForEvent(context.Background())
	require.NoError(err)
	require.Equal(commonEng.PendingTxs, msg)

	block2, err := vm.BuildBlockWithContext(context.Background(), validProposerCtx)
	require.NoError(err)

	// Require the block was built with a successful predicate result
	// Get internal block safely using unwrapBlock helper
	block2Internal := unwrapBlock(t, block2)
	ethBlock := block2Internal.ethBlock
	t.Logf("DEBUG TEST: block built at time=%d, numTxs=%d", ethBlock.Time(), len(ethBlock.Transactions()))
	headerPredicateResultsBytes := customheader.PredicateBytesFromExtra(ethBlock.Extra())
	results, err := predicate.ParseResults(headerPredicateResultsBytes)
	require.NoError(err)

	// Predicate results encode the index of invalid warp messages in a bitset.
	// An empty bitset indicates success.
	txResultsBytes := results.GetResults(
		getVerifiedWarpMessageTx.Hash(),
		warpcontract.ContractAddress,
	)
	bitset := set.BitsFromBytes(txResultsBytes)
	require.Zero(bitset.Len()) // Empty bitset indicates success

	block2VerifyWithCtx, ok := block2.(block.WithVerifyContext)
	require.True(ok)
	shouldVerifyWithCtx, err := block2VerifyWithCtx.ShouldVerifyWithContext(context.Background())
	require.NoError(err)
	require.True(shouldVerifyWithCtx)
	require.NoError(block2VerifyWithCtx.VerifyWithContext(context.Background(), validProposerCtx))
	require.NoError(vm.SetPreference(context.Background(), block2.ID()))

	// Verify the block with another valid context with identical predicate results
	require.NoError(block2VerifyWithCtx.VerifyWithContext(context.Background(), &block.Context{
		PChainHeight: minimumValidPChainHeight + 1,
	}))

	// Verify the block in a different context causing the warp message to fail verification changing
	// the expected header predicate results.
	require.ErrorIs(block2VerifyWithCtx.VerifyWithContext(context.Background(), &block.Context{
		PChainHeight: minimumValidPChainHeight - 1,
	}), errInvalidHeaderPredicateResults)

	// Accept the block after performing multiple VerifyWithContext operations
	require.NoError(block2.Accept(context.Background()))
	vm.blockChain.DrainAcceptorQueue()

	verifiedMessageReceipts := vm.blockChain.GetReceiptsByHash(ethBlock.Hash())
	t.Logf("DEBUG TEST: Getting receipts for block %d (hash=%s), got %d receipts", ethBlock.Number().Uint64(), ethBlock.Hash().Hex(), len(verifiedMessageReceipts))
	for i, r := range verifiedMessageReceipts {
		t.Logf("DEBUG TEST: Receipt[%d]: TxHash=%s, Status=%d", i, r.TxHash.Hex(), r.Status)
	}
	require.Len(verifiedMessageReceipts, 1)
	verifiedMessageTxReceipt := verifiedMessageReceipts[0]
	require.Equal(types.ReceiptStatusSuccessful, verifiedMessageTxReceipt.Status)

	expectedOutput, err := warpcontract.PackGetVerifiedWarpMessageOutput(warpcontract.GetVerifiedWarpMessageOutput{
		Message: warpcontract.WarpMessage{
			SourceChainID:       common.Hash(sourceChainID),
			OriginSenderAddress: testEthAddrs[0],
			Payload:             payloadData,
		},
		Valid: true,
	})
	require.NoError(err)

	tracerAPI := tracers.NewAPI(vm.eth.APIBackend)
	txTraceResults, err := tracerAPI.TraceBlockByHash(context.Background(), ethBlock.Hash(), nil)
	require.NoError(err)
	require.Len(txTraceResults, 1)
	blockTxTraceResultBytes, err := json.Marshal(txTraceResults[0].Result)
	require.NoError(err)
	unmarshalResults := make(map[string]interface{})
	require.NoError(json.Unmarshal(blockTxTraceResultBytes, &unmarshalResults))
	require.Equal("0x"+common.Bytes2Hex(expectedOutput), unmarshalResults["returnValue"])

	txTraceResult, err := tracerAPI.TraceTransaction(context.Background(), getVerifiedWarpMessageTx.Hash(), nil)
	require.NoError(err)
	txTraceResultBytes, err := json.Marshal(txTraceResult)
	require.NoError(err)
	require.JSONEq(string(txTraceResultBytes), string(blockTxTraceResultBytes))
}

func TestSignatureRequestsToVM(t *testing.T) {
	for _, scheme := range schemes {
		t.Run(scheme, func(t *testing.T) {
			testSignatureRequestsToVM(t, scheme)
		})
	}
}

func testSignatureRequestsToVM(t *testing.T, scheme string) {
	fork := "Latest"
	tvm := newVM(t, testVMConfig{
		fork:       &fork,
		configJSON: getConfig(scheme, ""),
	})

	defer func() {
		require.NoError(t, tvm.vm.Shutdown(context.Background()))
	}()

	// Setup known message
	knownPayload, err := payload.NewAddressedCall([]byte{0, 0, 0}, []byte("test"))
	require.NoError(t, err)
	chainID := consensuscontext.GetChainID(tvm.vm.ctx)
	knownWarpMessage, err := luxWarp.NewUnsignedMessage(consensuscontext.GetNetworkID(tvm.vm.ctx), chainID[:], knownPayload.Bytes())
	require.NoError(t, err)

	// Add the known message and get its signature to confirm
	require.NoError(t, tvm.vm.warpBackend.AddMessage(knownWarpMessage))
	knownMessageSignature, err := tvm.vm.warpBackend.GetMessageSignature(context.Background(), knownWarpMessage)
	require.NoError(t, err)

	// Setup known block
	lastAcceptedID, err := tvm.vm.LastAccepted(context.Background())
	require.NoError(t, err)
	knownBlockSignature, err := tvm.vm.warpBackend.GetBlockSignature(context.Background(), lastAcceptedID)
	require.NoError(t, err)

	type testCase struct {
		name             string
		message          *luxWarp.UnsignedMessage
		expectedResponse []byte
		err              *commonEng.AppError
	}

	tests := []testCase{
		{
			name:             "known message",
			message:          knownWarpMessage,
			expectedResponse: knownMessageSignature,
		},
		{
			name: "unknown message",
			message: func() *luxWarp.UnsignedMessage {
				unknownPayload, err := payload.NewAddressedCall([]byte{1, 1, 1}, []byte("unknown"))
				require.NoError(t, err)
				chainID := consensuscontext.GetChainID(tvm.vm.ctx)
				msg, err := luxWarp.NewUnsignedMessage(consensuscontext.GetNetworkID(tvm.vm.ctx), chainID[:], unknownPayload.Bytes())
				require.NoError(t, err)
				return msg
			}(),
			err: &commonEng.AppError{Code: warp.ParseErrCode},
		},
		{
			name: "known block",
			message: func() *luxWarp.UnsignedMessage {
				hashPayload, err := payload.NewHash(lastAcceptedID)
				require.NoError(t, err)
				chainID := consensuscontext.GetChainID(tvm.vm.ctx)
				msg, err := luxWarp.NewUnsignedMessage(consensuscontext.GetNetworkID(tvm.vm.ctx), chainID[:], hashPayload.Bytes())
				require.NoError(t, err)
				return msg
			}(),
			expectedResponse: knownBlockSignature,
		},
		{
			name: "unknown block",
			message: func() *luxWarp.UnsignedMessage {
				testID := ids.GenerateTestID()
				hashPayload, err := payload.NewHash(testID)
				require.NoError(t, err)
				chainID := consensuscontext.GetChainID(tvm.vm.ctx)
				msg, err := luxWarp.NewUnsignedMessage(consensuscontext.GetNetworkID(tvm.vm.ctx), chainID[:], hashPayload.Bytes())
				require.NoError(t, err)
				return msg
			}(),
			err: &commonEng.AppError{Code: warp.VerifyErrCode},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calledSendAppResponseFn := false
			calledSendAppErrorFn := false

			tvm.appSender.SendAppResponseF = func(ctx context.Context, nodeID ids.NodeID, requestID uint32, responseBytes []byte) error {
				calledSendAppResponseFn = true
				var response sdk.SignatureResponse
				if err := proto.Unmarshal(responseBytes, &response); err != nil {
					return err
				}
				require.Equal(t, test.expectedResponse, response.Signature)
				return nil
			}

			tvm.appSender.SendAppErrorF = func(ctx context.Context, nodeID ids.NodeID, requestID uint32, errCode int32, errString string) error {
				calledSendAppErrorFn = true
				require.ErrorIs(t, test.err, test.err)
				return nil
			}

			protoMsg := &sdk.SignatureRequest{Message: test.message.Bytes()}
			requestBytes, err := proto.Marshal(protoMsg)
			require.NoError(t, err)
			msg := p2p.PrefixMessage(p2p.ProtocolPrefix(lp118.HandlerID), requestBytes)

			// Send the app request and verify the response
			deadline := time.Now().Add(60 * time.Second)
			appErr := tvm.vm.Network.AppRequest(context.Background(), ids.GenerateTestNodeID(), 1, deadline, msg)
			require.Nil(t, appErr)
			if test.err != nil {
				require.True(t, calledSendAppErrorFn)
			} else {
				require.True(t, calledSendAppResponseFn)
			}
		})
	}
}

// noCloseDB wraps a database to ignore Close() calls, allowing reuse across VM restarts
type noCloseDB struct {
	luxfiDatabase.Database
}

func (db *noCloseDB) Close() error {
	return nil // Ignore close
}

func TestClearWarpDB(t *testing.T) {
	// Create a shared baseDB that persists across VM restarts
	// Wrap in noCloseDB to prevent the VM from closing it
	baseDB := &noCloseDB{memdb.New()}
	genesisJSON := toGenesisJSON(forkToChainConfig["Latest"])
	genesisBytes := []byte(genesisJSON)

	// First VM initialization
	ctx := createTestConsensusContext(t)
	db := prefixdb.New([]byte{1}, baseDB)
	vm := &VM{}
	require.NoError(t, vm.Initialize(context.Background(), ctx, db, genesisBytes, []byte{}, []byte{}, nil, nil, &TestSender{}))

	// use multiple messages to test that all messages get cleared
	payloads := [][]byte{[]byte("test1"), []byte("test2"), []byte("test3"), []byte("test4"), []byte("test5")}
	messages := []*luxWarp.UnsignedMessage{}

	// add all messages
	for _, msgPayload := range payloads {
		chainID := consensuscontext.GetChainID(vm.ctx)
		unsignedMsg, err := luxWarp.NewUnsignedMessage(consensuscontext.GetNetworkID(vm.ctx), chainID[:], msgPayload)
		require.NoError(t, err)
		require.NoError(t, vm.warpBackend.AddMessage(unsignedMsg))
		messages = append(messages, unsignedMsg)
	}

	require.NoError(t, vm.Shutdown(context.Background()))

	// Restart VM with the same database default should not prune the warp db
	vm = &VM{}
	// we need new context since the previous one has registered metrics.
	ctx = createTestConsensusContext(t)
	db = prefixdb.New([]byte{1}, baseDB) // Create new prefixdb from same baseDB
	require.NoError(t, vm.Initialize(context.Background(), ctx, db, genesisBytes, []byte{}, []byte{}, nil, nil, &TestSender{}))

	// check messages are still present
	for _, message := range messages {
		_, err := vm.warpBackend.GetMessageSignature(context.Background(), message)
		require.NoError(t, err)
	}

	require.NoError(t, vm.Shutdown(context.Background()))

	// restart the VM with pruning enabled
	vm = &VM{}
	config := `{"prune-warp-db-enabled": true}`
	ctx = createTestConsensusContext(t)
	db = prefixdb.New([]byte{1}, baseDB) // Create new prefixdb from same baseDB
	require.NoError(t, vm.Initialize(context.Background(), ctx, db, genesisBytes, []byte{}, []byte(config), nil, nil, &TestSender{}))

	it := vm.warpDB.NewIterator()
	require.False(t, it.Next())
	it.Release()

	// ensure all messages have been deleted
	for _, message := range messages {
		_, err := vm.warpBackend.GetMessageSignature(context.Background(), message)
		require.Error(t, err)
	}
}
