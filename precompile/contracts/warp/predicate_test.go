// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	agoUtils "github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/params"
	"github.com/luxfi/evm/precompile/precompileconfig"
	"github.com/luxfi/evm/precompile/testutils"
	"github.com/luxfi/evm/predicate"
	"github.com/luxfi/evm/utils"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/stretchr/testify/require"
)

const pChainHeight uint64 = 1337

var (
	_ agoUtils.Sortable[*testValidator] = (*testValidator)(nil)

	errTest        = errors.New("non-nil error")
	networkID      = uint32(54321)
	sourceChainID  = ids.GenerateTestID()
	sourceSubnetID = ids.GenerateTestID()

	// valid unsigned warp message used throughout testing
	unsignedMsg *interfaces.UnsignedMessage
	// valid addressed payload
	addressedPayload      *interfaces.AddressedCall
	addressedPayloadBytes []byte
	// blsSignatures of [unsignedMsg] from each of [testVdrs]
	blsSignatures []*interfaces.Signature

	numTestVdrs = 10_000
	testVdrs    []*testValidator
	vdrs        map[ids.NodeID]*interfaces.GetValidatorOutput

	predicateTests = make(map[string]testutils.PredicateTest)
)

func init() {
	testVdrs = make([]*testValidator, 0, numTestVdrs)
	for i := 0; i < numTestVdrs; i++ {
		testVdrs = append(testVdrs, newTestValidator())
	}
	agoUtils.Sort(testVdrs)

	vdrs = map[ids.NodeID]*interfaces.GetValidatorOutput{
		testVdrs[0].nodeID: {
			NodeID:    testVdrs[0].nodeID,
			PublicKey: testVdrs[0].vdr.PublicKey,
			Weight:    testVdrs[0].vdr.Weight,
		},
		testVdrs[1].nodeID: {
			NodeID:    testVdrs[1].nodeID,
			PublicKey: testVdrs[1].vdr.PublicKey,
			Weight:    testVdrs[1].vdr.Weight,
		},
		testVdrs[2].nodeID: {
			NodeID:    testVdrs[2].nodeID,
			PublicKey: testVdrs[2].vdr.PublicKey,
			Weight:    testVdrs[2].vdr.Weight,
		},
	}

	var err error
	addr := ids.GenerateTestShortID()
	addressedPayload, err = interfaces.NewAddressedCall(
		addr[:],
		[]byte{1, 2, 3},
	)
	if err != nil {
		panic(err)
	}
	addressedPayloadBytes = addressedPayload.Bytes()
	unsignedMsg, err = interfaces.NewUnsignedMessage(networkID, sourceChainID, addressedPayload.Bytes())
	if err != nil {
		panic(err)
	}

	for _, testVdr := range testVdrs {
		blsSignature, err := testVdr.sk.Sign(unsignedMsg.Bytes())
		if err != nil {
			panic(err)
		}
		blsSignatures = append(blsSignatures, blsSignature)
	}

	initWarpPredicateTests()
}

type testValidator struct {
	nodeID ids.NodeID
	sk     *interfaces.SecretKey
	vdr    *interfaces.Validator
	sk     interfaces.Signer
	vdr    *interfaces.Validator
}

func (v *testValidator) Compare(o *testValidator) int {
	return v.vdr.Compare(o.vdr)
}

func newTestValidator() *testValidator {
	sk, err := localsigner.New()
	if err != nil {
		panic(err)
	}

	nodeID := ids.GenerateTestNodeID()
	pk := sk.PublicKey()
	return &testValidator{
		nodeID: nodeID,
		sk:     sk,
		vdr: &interfaces.Validator{
			PublicKey:      pk,
			PublicKeyBytes: pk.Serialize(),
			Weight:         3,
			NodeIDs:        []ids.NodeID{nodeID},
		},
	}
}

type signatureTest struct {
	name      string
	stateF    func(*gomock.Controller) interfaces.State
	quorumNum uint64
	quorumDen uint64
	msgF      func(*require.Assertions) *interfaces.Message
	err       error
}

// createWarpMessage constructs a signed warp message using the global variable [unsignedMsg]
// and the first [numKeys] signatures from [blsSignatures]
func createWarpMessage(numKeys int) *interfaces.Message {
	aggregateSignature, err := interfaces.AggregateSignatures(blsSignatures[0:numKeys])
	if err != nil {
		panic(err)
	}
	bitSet := set.NewBits()
	for i := 0; i < numKeys; i++ {
		bitSet.Add(i)
	}
	warpSignature := &interfaces.BitSetSignature{
		Signers: bitSet.Bytes(),
	}
	copy(warpSignature.Signature[:], interfaces.SignatureToBytes(aggregateSignature))
	warpMsg, err := interfaces.NewMessage(unsignedMsg, warpSignature)
	if err != nil {
		panic(err)
	}
	return warpMsg
}

// createPredicate constructs a warp message using createWarpMessage with numKeys signers
// and packs it into predicate encoding.
func createPredicate(numKeys int) []byte {
	warpMsg := createWarpMessage(numKeys)
	predicateBytes := predicate.PackPredicate(warpMsg.Bytes())
	return predicateBytes
}

// validatorRange specifies a range of validators to include from [start, end), a staking weight
// to specify for each validator in that range, and whether or not to include the public key.
type validatorRange struct {
	start     int
	end       int
	weight    uint64
	publicKey bool
}

// createConsensusCtx creates a consensus.Context instance with a validator state specified by the given validatorRanges
func createConsensusCtx(validatorRanges []validatorRange) *consensus.Context {
	getValidatorsOutput := make(map[ids.NodeID]*interfaces.GetValidatorOutput)

	for _, validatorRange := range validatorRanges {
		for i := validatorRange.start; i < validatorRange.end; i++ {
			validatorOutput := &interfaces.GetValidatorOutput{
				NodeID: testVdrs[i].nodeID,
				Weight: validatorRange.weight,
			}
			if validatorRange.publicKey {
				validatorOutput.PublicKey = testVdrs[i].vdr.PublicKey
			}
			getValidatorsOutput[testVdrs[i].nodeID] = validatorOutput
		}
	}

	consensusCtx := utils.TestConsensusContext()
	state := &validatorstest.State{
		GetSubnetIDF: func(ctx context.Context, chainID ids.ID) (ids.ID, error) {
			return sourceSubnetID, nil
		},
		GetValidatorSetF: func(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*interfaces.GetValidatorOutput, error) {
			return getValidatorsOutput, nil
		},
	}
	consensusCtx.ValidatorState = state
	consensusCtx.NetworkID = networkID
	return consensusCtx
}

func createValidPredicateTest(consensusCtx *consensus.Context, numKeys uint64, predicateBytes []byte) testutils.PredicateTest {
	return testutils.PredicateTest{
		Config: NewDefaultConfig(utils.NewUint64(0)),
		PredicateContext: &precompileinterfaces.PredicateContext{
			ConsensusCtx: consensusCtx,
			ProposerVMBlockCtx: &interfaces.Context{
				PChainHeight: 1,
			},
		},
		PredicateBytes: predicateBytes,
		Gas:            GasCostPerSignatureVerification + uint64(len(predicateBytes))*GasCostPerWarpMessageBytes + numKeys*GasCostPerWarpSigner,
		GasErr:         nil,
		ExpectedErr:    nil,
	}
}

func TestWarpMessageFromPrimaryNetwork(t *testing.T) {
	for _, requirePrimaryNetworkSigners := range []bool{true, false} {
		testWarpMessageFromPrimaryNetwork(t, requirePrimaryNetworkSigners)
	}
}

func testWarpMessageFromPrimaryNetwork(t *testing.T, requirePrimaryNetworkSigners bool) {
	require := require.New(t)
	numKeys := 10
	cChainID := ids.GenerateTestID()
	addressedCall, err := interfaces.NewAddressedCall(agoUtils.RandomBytes(20), agoUtils.RandomBytes(100))
	require.NoError(err)
	unsignedMsg, err := interfaces.NewUnsignedMessage(networkID, cChainID, addressedCall.Bytes())
	require.NoError(err)

	getValidatorsOutput := make(map[ids.NodeID]*interfaces.GetValidatorOutput)
	blsSignatures := make([]*interfaces.Signature, 0, numKeys)
	for i := 0; i < numKeys; i++ {
		sig, err := testVdrs[i].sk.Sign(unsignedMsg.Bytes())
		require.NoError(err)

		validatorOutput := &interfaces.GetValidatorOutput{
			NodeID:    testVdrs[i].nodeID,
			Weight:    20,
			PublicKey: testVdrs[i].vdr.PublicKey,
		}
		getValidatorsOutput[testVdrs[i].nodeID] = validatorOutput
		blsSignatures = append(blsSignatures, sig)
	}
	aggregateSignature, err := interfaces.AggregateSignatures(blsSignatures)
	require.NoError(err)
	bitSet := set.NewBits()
	for i := 0; i < numKeys; i++ {
		bitSet.Add(i)
	}
	warpSignature := &interfaces.BitSetSignature{
		Signers: bitSet.Bytes(),
	}
	copy(warpSignature.Signature[:], interfaces.SignatureToBytes(aggregateSignature))
	warpMsg, err := interfaces.NewMessage(unsignedMsg, warpSignature)
	require.NoError(err)

	predicateBytes := predicate.PackPredicate(warpMsg.Bytes())

	consensusCtx := utils.TestConsensusContext()
	consensusCtx.SubnetID = ids.GenerateTestID()
	consensusCtx.ChainID = ids.GenerateTestID()
	consensusCtx.CChainID = cChainID
	consensusCtx.NetworkID = networkID
	consensusCtx.ValidatorState = &validatorstest.State{
		GetSubnetIDF: func(ctx context.Context, chainID ids.ID) (ids.ID, error) {
			require.Equal(chainID, cChainID)
			return constants.PrimaryNetworkID, nil // Return Primary Network SubnetID
		},
		GetValidatorSetF: func(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*interfaces.GetValidatorOutput, error) {
			expectedSubnetID := consensusCtx.SubnetID
			if requirePrimaryNetworkSigners {
				expectedSubnetID = constants.PrimaryNetworkID
			}
			require.Equal(expectedSubnetID, subnetID)
			return getValidatorsOutput, nil
		},
	}

	test := testutils.PredicateTest{
		Config: NewConfig(utils.NewUint64(0), 0, requirePrimaryNetworkSigners),
		PredicateContext: &precompileinterfaces.PredicateContext{
			ConsensusCtx: consensusCtx,
			ProposerVMBlockCtx: &interfaces.Context{
				PChainHeight: 1,
			},
		},
		PredicateBytes: predicateBytes,
		Gas:            GasCostPerSignatureVerification + uint64(len(predicateBytes))*GasCostPerWarpMessageBytes + uint64(numKeys)*GasCostPerWarpSigner,
		GasErr:         nil,
		ExpectedErr:    nil,
	}

	test.Run(t)
}

func TestInvalidPredicatePacking(t *testing.T) {
	numKeys := 1
	consensusCtx := createConsensusCtx([]validatorRange{
		{
			start:     0,
			end:       numKeys,
			weight:    20,
			publicKey: true,
		},
	})
	predicateBytes := createPredicate(numKeys)
	predicateBytes = append(predicateBytes, byte(0x01)) // Invalidate the predicate byte packing

	test := testutils.PredicateTest{
		Config: NewDefaultConfig(utils.NewUint64(0)),
		PredicateContext: &precompileinterfaces.PredicateContext{
			ConsensusCtx: consensusCtx,
			ProposerVMBlockCtx: &interfaces.Context{
				PChainHeight: 1,
			},
		},
		PredicateBytes: predicateBytes,
		Gas:            GasCostPerSignatureVerification + uint64(len(predicateBytes))*GasCostPerWarpMessageBytes + uint64(numKeys)*GasCostPerWarpSigner,
		GasErr:         errInvalidPredicateBytes,
	}

	test.Run(t)
}

func TestInvalidWarpMessage(t *testing.T) {
	numKeys := 1
	consensusCtx := createConsensusCtx([]validatorRange{
		{
			start:     0,
			end:       numKeys,
			weight:    20,
			publicKey: true,
		},
	})
	warpMsg := createWarpMessage(1)
	warpMsgBytes := warpMsg.Bytes()
	warpMsgBytes = append(warpMsgBytes, byte(0x01)) // Invalidate warp message packing
	predicateBytes := predicate.PackPredicate(warpMsgBytes)

	test := testutils.PredicateTest{
		Config: NewDefaultConfig(utils.NewUint64(0)),
		PredicateContext: &precompileinterfaces.PredicateContext{
			ConsensusCtx: consensusCtx,
			ProposerVMBlockCtx: &interfaces.Context{
				PChainHeight: 1,
			},
		},
		PredicateBytes: predicateBytes,
		Gas:            GasCostPerSignatureVerification + uint64(len(predicateBytes))*GasCostPerWarpMessageBytes + uint64(numKeys)*GasCostPerWarpSigner,
		GasErr:         errInvalidWarpMsg,
	}

	test.Run(t)
}

func TestInvalidAddressedPayload(t *testing.T) {
	numKeys := 1
	consensusCtx := createConsensusCtx([]validatorRange{
		{
			start:     0,
			end:       numKeys,
			weight:    20,
			publicKey: true,
		},
	})
	aggregateSignature, err := interfaces.AggregateSignatures(blsSignatures[0:numKeys])
	require.NoError(t, err)
	bitSet := set.NewBits()
	for i := 0; i < numKeys; i++ {
		bitSet.Add(i)
	}
	warpSignature := &interfaces.BitSetSignature{
		Signers: bitSet.Bytes(),
	}
	copy(warpSignature.Signature[:], interfaces.SignatureToBytes(aggregateSignature))
	// Create an unsigned message with an invalid addressed payload
	unsignedMsg, err := interfaces.NewUnsignedMessage(networkID, sourceChainID, []byte{1, 2, 3})
	require.NoError(t, err)
	warpMsg, err := interfaces.NewMessage(unsignedMsg, warpSignature)
	require.NoError(t, err)
	warpMsgBytes := warpMsg.Bytes()
	predicateBytes := predicate.PackPredicate(warpMsgBytes)

	test := testutils.PredicateTest{
		Config: NewDefaultConfig(utils.NewUint64(0)),
		PredicateContext: &precompileinterfaces.PredicateContext{
			ConsensusCtx: consensusCtx,
			ProposerVMBlockCtx: &interfaces.Context{
				PChainHeight: 1,
			},
		},
		PredicateBytes: predicateBytes,
		Gas:            GasCostPerSignatureVerification + uint64(len(predicateBytes))*GasCostPerWarpMessageBytes + uint64(numKeys)*GasCostPerWarpSigner,
		GasErr:         errInvalidWarpMsgPayload,
	}

	test.Run(t)
}

func TestInvalidBitSet(t *testing.T) {
	addressedCall, err := interfaces.NewAddressedCall(agoUtils.RandomBytes(20), agoUtils.RandomBytes(100))
	require.NoError(t, err)
	unsignedMsg, err := interfaces.NewUnsignedMessage(
		networkID,
		sourceChainID,
		addressedCall.Bytes(),
	)
	require.NoError(t, err)

	msg, err := interfaces.NewMessage(
		unsignedMsg,
		&interfaces.BitSetSignature{
			Signers:   make([]byte, 1),
			Signature: [interfaces.SignatureLen]byte{},
		},
	)
	require.NoError(t, err)

	numKeys := 1
	consensusCtx := createConsensusCtx([]validatorRange{
		{
			start:     0,
			end:       numKeys,
			weight:    20,
			publicKey: true,
		},
	})
	predicateBytes := predicate.PackPredicate(msg.Bytes())
	test := testutils.PredicateTest{
		Config: NewDefaultConfig(utils.NewUint64(0)),
		PredicateContext: &precompileinterfaces.PredicateContext{
			ConsensusCtx: consensusCtx,
			ProposerVMBlockCtx: &interfaces.Context{
				PChainHeight: 1,
			},
		},
		PredicateBytes: predicateBytes,
		Gas:            GasCostPerSignatureVerification + uint64(len(predicateBytes))*GasCostPerWarpMessageBytes + uint64(numKeys)*GasCostPerWarpSigner,
		GasErr:         errCannotGetNumSigners,
	}

	test.Run(t)
}

func TestWarpSignatureWeightsDefaultQuorumNumerator(t *testing.T) {
	consensusCtx := createConsensusCtx([]validatorRange{
		{
			start:     0,
			end:       100,
			weight:    20,
			publicKey: true,
		},
	})

	tests := make(map[string]testutils.PredicateTest)
	for _, numSigners := range []int{
		1,
		int(WarpDefaultQuorumNumerator) - 1,
		int(WarpDefaultQuorumNumerator),
		int(WarpDefaultQuorumNumerator) + 1,
		int(WarpQuorumDenominator) - 1,
		int(WarpQuorumDenominator),
		int(WarpQuorumDenominator) + 1,
	} {
		predicateBytes := createPredicate(numSigners)
		// The predicate is valid iff the number of signers is >= the required numerator and does not exceed the denominator.
		var expectedErr error
		if numSigners >= int(WarpDefaultQuorumNumerator) && numSigners <= int(WarpQuorumDenominator) {
			expectedErr = nil
		} else {
			expectedErr = errFailedVerification
		}

		tests[fmt.Sprintf("default quorum %d signature(s)", numSigners)] = testutils.PredicateTest{
			Config: NewDefaultConfig(utils.NewUint64(0)),
			PredicateContext: &precompileinterfaces.PredicateContext{
				ConsensusCtx: consensusCtx,
				ProposerVMBlockCtx: &interfaces.Context{
					PChainHeight: 1,
				},
			},
			PredicateBytes: predicateBytes,
			Gas:            GasCostPerSignatureVerification + uint64(len(predicateBytes))*GasCostPerWarpMessageBytes + uint64(numSigners)*GasCostPerWarpSigner,
			GasErr:         nil,
			ExpectedErr:    expectedErr,
		}
	}
	testutils.RunPredicateTests(t, tests)
}

// multiple messages all correct, multiple messages all incorrect, mixed bag
func TestWarpMultiplePredicates(t *testing.T) {
	consensusCtx := createConsensusCtx([]validatorRange{
		{
			start:     0,
			end:       100,
			weight:    20,
			publicKey: true,
		},
	})

	tests := make(map[string]testutils.PredicateTest)
	for _, validMessageIndices := range [][]bool{
		{},
		{true, false},
		{false, true},
		{false, false},
		{true, true},
	} {
		var (
			numSigners            = int(WarpQuorumDenominator)
			invalidPredicateBytes = createPredicate(1)
			validPredicateBytes   = createPredicate(numSigners)
		)

		for _, valid := range validMessageIndices {
			var (
				predicate   []byte
				expectedGas uint64
				expectedErr error
			)
			if valid {
				predicate = validPredicateBytes
				expectedGas = GasCostPerSignatureVerification + uint64(len(validPredicateBytes))*GasCostPerWarpMessageBytes + uint64(numSigners)*GasCostPerWarpSigner
				expectedErr = nil
			} else {
				expectedGas = GasCostPerSignatureVerification + uint64(len(invalidPredicateBytes))*GasCostPerWarpMessageBytes + uint64(1)*GasCostPerWarpSigner
				predicate = invalidPredicateBytes
				expectedErr = errFailedVerification
			}

			tests[fmt.Sprintf("multiple predicates %v", validMessageIndices)] = testutils.PredicateTest{
				Config: NewDefaultConfig(utils.NewUint64(0)),
				PredicateContext: &precompileinterfaces.PredicateContext{
					ConsensusCtx: consensusCtx,
					ProposerVMBlockCtx: &interfaces.Context{
						PChainHeight: 1,
					},
				},
				PredicateBytes: predicate,
				Gas:            expectedGas,
				GasErr:         nil,
				ExpectedErr:    expectedErr,
			}
		}
	}
	testutils.RunPredicateTests(t, tests)
}

func TestWarpSignatureWeightsNonDefaultQuorumNumerator(t *testing.T) {
	consensusCtx := createConsensusCtx([]validatorRange{
		{
			start:     0,
			end:       100,
			weight:    20,
			publicKey: true,
		},
	})

	tests := make(map[string]testutils.PredicateTest)
	nonDefaultQuorumNumerator := 50
	// Ensure this test fails if the DefaultQuroumNumerator is changed to an unexpected value during development
	require.NotEqual(t, nonDefaultQuorumNumerator, int(WarpDefaultQuorumNumerator))
	// Add cases with default quorum
	for _, numSigners := range []int{nonDefaultQuorumNumerator, nonDefaultQuorumNumerator + 1, 99, 100, 101} {
		predicateBytes := createPredicate(numSigners)
		// The predicate is valid iff the number of signers is >= the required numerator and does not exceed the denominator.
		var expectedErr error
		if numSigners >= nonDefaultQuorumNumerator && numSigners <= int(WarpQuorumDenominator) {
			expectedErr = nil
		} else {
			expectedErr = errFailedVerification
		}

		name := fmt.Sprintf("non-default quorum %d signature(s)", numSigners)
		tests[name] = testutils.PredicateTest{
			Config: NewConfig(utils.NewUint64(0), uint64(nonDefaultQuorumNumerator), false),
			PredicateContext: &precompileinterfaces.PredicateContext{
				ConsensusCtx: consensusCtx,
				ProposerVMBlockCtx: &interfaces.Context{
					PChainHeight: 1,
				},
			},
			PredicateBytes: predicateBytes,
			Gas:            GasCostPerSignatureVerification + uint64(len(predicateBytes))*GasCostPerWarpMessageBytes + uint64(numSigners)*GasCostPerWarpSigner,
			GasErr:         nil,
			ExpectedErr:    expectedErr,
		}
	}

	testutils.RunPredicateTests(t, tests)
}

func initWarpPredicateTests() {
	for _, totalNodes := range []int{10, 100, 1_000, 10_000} {
		testName := fmt.Sprintf("%d signers/%d validators", totalNodes, totalNodes)

		predicateBytes := createPredicate(totalNodes)
		consensusCtx := createConsensusCtx([]validatorRange{
			{
				start:     0,
				end:       totalNodes,
				weight:    20,
				publicKey: true,
			},
		})
		predicateTests[testName] = createValidPredicateTest(consensusCtx, uint64(totalNodes), predicateBytes)
	}

	numSigners := 10
	for _, totalNodes := range []int{100, 1_000, 10_000} {
		testName := fmt.Sprintf("%d signers (heavily weighted)/%d validators", numSigners, totalNodes)

		predicateBytes := createPredicate(numSigners)
		consensusCtx := createConsensusCtx([]validatorRange{
			{
				start:     0,
				end:       numSigners,
				weight:    10_000_000,
				publicKey: true,
			},
			{
				start:     numSigners,
				end:       totalNodes,
				weight:    20,
				publicKey: true,
			},
		})
		predicateTests[testName] = createValidPredicateTest(consensusCtx, uint64(numSigners), predicateBytes)
	}

	for _, totalNodes := range []int{100, 1_000, 10_000} {
		testName := fmt.Sprintf("%d signers (heavily weighted)/%d validators (non-signers without registered PublicKey)", numSigners, totalNodes)

		predicateBytes := createPredicate(numSigners)
		consensusCtx := createConsensusCtx([]validatorRange{
			{
				start:     0,
				end:       numSigners,
				weight:    10_000_000,
				publicKey: true,
			},
			{
				start:     numSigners,
				end:       totalNodes,
				weight:    20,
				publicKey: false,
			},
		})
		predicateTests[testName] = createValidPredicateTest(consensusCtx, uint64(numSigners), predicateBytes)
	}

	for _, totalNodes := range []int{100, 1_000, 10_000} {
		testName := fmt.Sprintf("%d validators w/ %d signers/repeated PublicKeys", totalNodes, numSigners)

		predicateBytes := createPredicate(numSigners)
		getValidatorsOutput := make(map[ids.NodeID]*interfaces.GetValidatorOutput, totalNodes)
		for i := 0; i < totalNodes; i++ {
			getValidatorsOutput[testVdrs[i].nodeID] = &interfaces.GetValidatorOutput{
				NodeID:    testVdrs[i].nodeID,
				Weight:    20,
				PublicKey: testVdrs[i%numSigners].vdr.PublicKey,
			}
		}

		consensusCtx := utils.TestConsensusContext()
		consensusCtx.NetworkID = networkID
		state := &validatorstest.State{
			GetSubnetIDF: func(ctx context.Context, chainID ids.ID) (ids.ID, error) {
				return sourceSubnetID, nil
			},
			GetValidatorSetF: func(ctx context.Context, height uint64, subnetID ids.ID) (map[ids.NodeID]*interfaces.GetValidatorOutput, error) {
				return getValidatorsOutput, nil
			},
		}
		consensusCtx.ValidatorState = state

		predicateTests[testName] = createValidPredicateTest(consensusCtx, uint64(numSigners), predicateBytes)
	}
}

func TestWarpPredicate(t *testing.T) {
	testutils.RunPredicateTests(t, predicateTests)
}

func BenchmarkWarpPredicate(b *testing.B) {
	testutils.RunPredicateBenchmarks(b, predicateTests)
}
