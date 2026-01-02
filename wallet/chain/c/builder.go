// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package c

import (
	"context"
	"errors"
	"math/big"

	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/node/utils"
	"github.com/luxfi/node/utils/math"
	"github.com/luxfi/node/vms/components/lux"
	"github.com/luxfi/node/vms/secp256k1fx"
	"github.com/luxfi/sdk/wallet/primary/common"

	ethcommon "github.com/luxfi/geth/common"
)

const luxConversionRateInt = 1_000_000_000

var (
	_ Builder = (*builder)(nil)

	errInsufficientFunds = errors.New("insufficient funds")

	// luxConversionRate is the conversion rate between the smallest
	// denomination on the X-Chain and P-chain, 1 nLUX, and the smallest
	// denomination on the C-Chain 1 wei. Where 1 nLUX = 1 gWei.
	//
	// This is only required for LUX because the denomination of 1 LUX is 9
	// decimal places on the X and P chains, but is 18 decimal places within the
	// EVM.
	luxConversionRate = big.NewInt(luxConversionRateInt)
)

// Builder provides a convenient interface for building unsigned C-chain
// transactions.
type Builder interface {
	// Context returns the configuration of the chain that this builder uses to
	// create transactions.
	Context() *Context

	// GetBalance calculates the amount of LUX that this builder has control
	// over.
	GetBalance(
		options ...common.Option,
	) (*big.Int, error)

	// GetImportableBalance calculates the amount of LUX that this builder
	// could import from the provided chain.
	//
	// - [chainID] specifies the chain the funds are from.
	GetImportableBalance(
		chainID ids.ID,
		options ...common.Option,
	) (uint64, error)

	// NewImportTx creates an import transaction that attempts to consume all
	// the available UTXOs and import the funds to [to].
	//
	// - [chainID] specifies the chain to be importing funds from.
	// - [to] specifies where to send the imported funds to.
	// - [baseFee] specifies the fee price willing to be paid by this tx.
	NewImportTx(
		chainID ids.ID,
		to ethcommon.Address,
		baseFee *big.Int,
		options ...common.Option,
	) (*UnsignedImportTx, error)

	// NewExportTx creates an export transaction that attempts to send all the
	// provided [outputs] to the requested [chainID].
	//
	// - [chainID] specifies the chain to be exporting the funds to.
	// - [outputs] specifies the outputs to send to the [chainID].
	// - [baseFee] specifies the fee price willing to be paid by this tx.
	NewExportTx(
		chainID ids.ID,
		outputs []*secp256k1fx.TransferOutput,
		baseFee *big.Int,
		options ...common.Option,
	) (*UnsignedExportTx, error)
}

// BuilderBackend specifies the required information needed to build unsigned
// C-chain transactions.
type BuilderBackend interface {
	UTXOs(ctx context.Context, sourceChainID ids.ID) ([]*lux.UTXO, error)
	Balance(ctx context.Context, addr ethcommon.Address) (*big.Int, error)
	Nonce(ctx context.Context, addr ethcommon.Address) (uint64, error)
}

type builder struct {
	luxAddrs set.Set[ids.ShortID]
	ethAddrs set.Set[ethcommon.Address]
	context  *Context
	backend  BuilderBackend
}

// NewBuilder returns a new transaction builder.
//
//   - [luxAddrs] is the set of addresses in the LUX format that the builder
//     assumes can be used when signing the transactions in the future.
//   - [ethAddrs] is the set of addresses in the Eth format that the builder
//     assumes can be used when signing the transactions in the future.
//   - [backend] provides the required access to the chain's context and state
//     to build out the transactions.
func NewBuilder(
	luxAddrs set.Set[ids.ShortID],
	ethAddrs set.Set[ethcommon.Address],
	context *Context,
	backend BuilderBackend,
) Builder {
	return &builder{
		luxAddrs: luxAddrs,
		ethAddrs: ethAddrs,
		context:  context,
		backend:  backend,
	}
}

func (b *builder) Context() *Context {
	return b.context
}

func (b *builder) GetBalance(
	options ...common.Option,
) (*big.Int, error) {
	var (
		ops          = common.NewOptions(options)
		ctx          = ops.Context()
		addrs        = ops.EthAddresses(b.ethAddrs)
		totalBalance = new(big.Int)
	)
	for addr := range addrs {
		balance, err := b.backend.Balance(ctx, addr)
		if err != nil {
			return nil, err
		}
		totalBalance.Add(totalBalance, balance)
	}

	return totalBalance, nil
}

func (b *builder) GetImportableBalance(
	chainID ids.ID,
	options ...common.Option,
) (uint64, error) {
	ops := common.NewOptions(options)
	utxos, err := b.backend.UTXOs(ops.Context(), chainID)
	if err != nil {
		return 0, err
	}

	var (
		addrs           = ops.Addresses(b.luxAddrs)
		minIssuanceTime = ops.MinIssuanceTime()
		luxAssetID      = b.context.XAssetID
		balance         uint64
	)
	for _, utxo := range utxos {
		amount, _, ok := getSpendableAmount(utxo, addrs, minIssuanceTime, luxAssetID)
		if !ok {
			continue
		}

		newBalance, err := math.Add(balance, amount)
		if err != nil {
			return 0, err
		}
		balance = newBalance
	}

	return balance, nil
}

func (b *builder) NewImportTx(
	chainID ids.ID,
	to ethcommon.Address,
	baseFee *big.Int,
	options ...common.Option,
) (*UnsignedImportTx, error) {
	ops := common.NewOptions(options)
	utxos, err := b.backend.UTXOs(ops.Context(), chainID)
	if err != nil {
		return nil, err
	}

	var (
		addrs           = ops.Addresses(b.luxAddrs)
		minIssuanceTime = ops.MinIssuanceTime()
		luxAssetID      = b.context.XAssetID

		importedInputs = make([]*lux.TransferableInput, 0, len(utxos))
		importedAmount uint64
	)
	for _, utxo := range utxos {
		amount, inputSigIndices, ok := getSpendableAmount(utxo, addrs, minIssuanceTime, luxAssetID)
		if !ok {
			continue
		}

		importedInputs = append(importedInputs, &lux.TransferableInput{
			UTXOID: utxo.UTXOID,
			Asset:  utxo.Asset,
			FxID:   secp256k1fx.ID,
			In: &secp256k1fx.TransferInput{
				Amt: amount,
				Input: secp256k1fx.Input{
					SigIndices: inputSigIndices,
				},
			},
		})

		newImportedAmount, err := math.Add(importedAmount, amount)
		if err != nil {
			return nil, err
		}
		importedAmount = newImportedAmount
	}

	utils.Sort(importedInputs)
	tx := &UnsignedImportTx{
		BaseTx: BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.context.BlockchainID,
		},
		SourceChain:    chainID,
		ImportedInputs: importedInputs,
	}

	// We must initialize the bytes of the tx to calculate the initial cost
	wrappedTx := &Tx{UnsignedAtomicTx: tx}
	if err := wrappedTx.Sign(Codec, nil); err != nil {
		return nil, err
	}

	gasUsedWithoutOutput, err := tx.GasUsed(true /*=IsApricotPhase5*/)
	if err != nil {
		return nil, err
	}
	gasUsedWithOutput := gasUsedWithoutOutput + EVMOutputGas

	txFee, err := CalculateDynamicFee(gasUsedWithOutput, baseFee)
	if err != nil {
		return nil, err
	}

	if importedAmount <= txFee {
		return nil, errInsufficientFunds
	}

	tx.Outs = []*EVMOutput{{
		Address: to,
		Amount:  importedAmount - txFee,
	}}
	return tx, nil
}

func (b *builder) NewExportTx(
	chainID ids.ID,
	outputs []*secp256k1fx.TransferOutput,
	baseFee *big.Int,
	options ...common.Option,
) (*UnsignedExportTx, error) {
	var (
		luxAssetID      = b.context.XAssetID
		exportedOutputs = make([]*lux.TransferableOutput, len(outputs))
		exportedAmount  uint64
	)
	for i, output := range outputs {
		exportedOutputs[i] = &lux.TransferableOutput{
			Asset: lux.Asset{ID: luxAssetID},
			FxID:  secp256k1fx.ID,
			Out:   output,
		}

		newExportedAmount, err := math.Add(exportedAmount, output.Amt)
		if err != nil {
			return nil, err
		}
		exportedAmount = newExportedAmount
	}

	lux.SortTransferableOutputs(exportedOutputs, Codec)
	tx := &UnsignedExportTx{
		BaseTx: BaseTx{
			NetworkID:    b.context.NetworkID,
			BlockchainID: b.context.BlockchainID,
		},
		DestinationChain: chainID,
		ExportedOutputs:  exportedOutputs,
	}

	// We must initialize the bytes of the tx to calculate the initial cost
	wrappedTx := &Tx{UnsignedAtomicTx: tx}
	if err := wrappedTx.Sign(Codec, nil); err != nil {
		return nil, err
	}

	cost, err := tx.GasUsed(true /*=IsApricotPhase5*/)
	if err != nil {
		return nil, err
	}

	initialFee, err := CalculateDynamicFee(cost, baseFee)
	if err != nil {
		return nil, err
	}

	amountToConsume, err := math.Add(exportedAmount, initialFee)
	if err != nil {
		return nil, err
	}

	var (
		ops    = common.NewOptions(options)
		ctx    = ops.Context()
		addrs  = ops.EthAddresses(b.ethAddrs)
		inputs = make([]*EVMInput, 0, addrs.Len())
	)
	for addr := range addrs {
		if amountToConsume == 0 {
			break
		}

		prevFee, err := CalculateDynamicFee(cost, baseFee)
		if err != nil {
			return nil, err
		}

		newCost := cost + EVMInputGas
		newFee, err := CalculateDynamicFee(newCost, baseFee)
		if err != nil {
			return nil, err
		}

		additionalFee := newFee - prevFee

		balance, err := b.backend.Balance(ctx, addr)
		if err != nil {
			return nil, err
		}

		// Since the asset is LUX, we divide by the luxConversionRate to
		// convert back to the correct denomination of LUX that can be
		// exported.
		luxBalance := new(big.Int).Div(balance, luxConversionRate).Uint64()

		// If the balance for [addr] is insufficient to cover the additional
		// cost of adding an input to the transaction, skip adding the input
		// altogether.
		if luxBalance <= additionalFee {
			continue
		}

		// Update the cost for the next iteration
		cost = newCost

		amountToConsume, err = math.Add(amountToConsume, additionalFee)
		if err != nil {
			return nil, err
		}

		nonce, err := b.backend.Nonce(ctx, addr)
		if err != nil {
			return nil, err
		}

		inputAmount := min(amountToConsume, luxBalance)
		inputs = append(inputs, &EVMInput{
			Address: addr,
			Amount:  inputAmount,
			AssetID: luxAssetID,
			Nonce:   nonce,
		})
		amountToConsume -= inputAmount
	}

	if amountToConsume > 0 {
		return nil, errInsufficientFunds
	}

	utils.Sort(inputs)
	tx.Ins = inputs

	consensusCtx, err := newConsensusContext(b.context)
	if err != nil {
		return nil, err
	}
	for _, out := range tx.ExportedOutputs {
		out.InitCtx(consensusCtx)
	}
	return tx, nil
}

func getSpendableAmount(
	utxo *lux.UTXO,
	addrs set.Set[ids.ShortID],
	minIssuanceTime uint64,
	luxAssetID ids.ID,
) (uint64, []uint32, bool) {
	if utxo.Asset.ID != luxAssetID {
		// Only LUX can be imported
		return 0, nil, false
	}

	out, ok := utxo.Out.(*secp256k1fx.TransferOutput)
	if !ok {
		// Can't import an unknown transfer output type
		return 0, nil, false
	}

	inputSigIndices, ok := common.MatchOwners(&out.OutputOwners, addrs, minIssuanceTime)
	return out.Amt, inputSigIndices, ok
}
