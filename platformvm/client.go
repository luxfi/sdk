// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"context"
	"time"

	"github.com/luxfi/address"
	json "github.com/luxfi/codec/jsonrpc"
	"github.com/luxfi/constants"
	"github.com/luxfi/crypto/bls"
	"github.com/luxfi/formatting"
	"github.com/luxfi/ids"
	"github.com/luxfi/protocol/p/fx"
	"github.com/luxfi/protocol/p/status"
	"github.com/luxfi/protocol/p/validators/fee"
	"github.com/luxfi/rpc"
	"github.com/luxfi/utxo/secp256k1fx"
	validators "github.com/luxfi/validators"
	"github.com/luxfi/vm/api"
	"github.com/luxfi/vm/components/gas"

	platformapi "github.com/luxfi/protocol/p/api"
)

type Client struct {
	Requester rpc.EndpointRequester
	networkID uint32
}

func NewClient(uri string) *Client {
	return &Client{Requester: rpc.NewEndpointRequester(
		uri + "/ext/P",
	)}
}

// NewClientWithNetworkID returns a new platformvm.Client with the network ID set
// for proper bech32 address formatting
func NewClientWithNetworkID(uri string, networkID uint32) *Client {
	return &Client{
		Requester: rpc.NewEndpointRequester(uri + "/ext/P"),
		networkID: networkID,
	}
}

// SetNetworkID sets the network ID for address formatting
func (c *Client) SetNetworkID(networkID uint32) {
	c.networkID = networkID
}

// formatAddresses converts ShortIDs to bech32 P-Chain addresses
func (c *Client) formatAddresses(addrs []ids.ShortID) ([]string, error) {
	hrp := constants.GetHRP(c.networkID)
	formatted := make([]string, len(addrs))
	for i, addr := range addrs {
		addrStr, err := address.Format("P", hrp, addr[:])
		if err != nil {
			return nil, err
		}
		formatted[i] = addrStr
	}
	return formatted, nil
}

// GetHeight returns the current block height.
func (c *Client) GetHeight(ctx context.Context, options ...rpc.Option) (uint64, error) {
	res := &api.GetHeightResponse{}
	err := c.Requester.SendRequest(ctx, "platform.getHeight", struct{}{}, res, options...)
	return uint64(res.Height), err
}

// GetProposedHeight returns the current height of this node's proposer VM.
func (c *Client) GetProposedHeight(ctx context.Context, options ...rpc.Option) (uint64, error) {
	res := &api.GetHeightResponse{}
	err := c.Requester.SendRequest(ctx, "platform.getProposedHeight", struct{}{}, res, options...)
	return uint64(res.Height), err
}

// GetBalance returns the balance of addrs.
//
// Deprecated: GetUTXOs should be used instead.
func (c *Client) GetBalance(ctx context.Context, addrs []ids.ShortID, options ...rpc.Option) (*GetBalanceResponse, error) {
	res := &GetBalanceResponse{}
	err := c.Requester.SendRequest(ctx, "platform.getBalance", &GetBalanceRequest{
		Addresses: ids.ShortIDsToStrings(addrs),
	}, res, options...)
	return res, err
}

// GetUTXOs returns the byte representation of the UTXOs controlled by addrs.
func (c *Client) GetUTXOs(
	ctx context.Context,
	addrs []ids.ShortID,
	limit uint32,
	startAddress ids.ShortID,
	startUTXOID ids.ID,
	options ...rpc.Option,
) ([][]byte, ids.ShortID, ids.ID, error) {
	return c.GetAtomicUTXOs(ctx, addrs, "", limit, startAddress, startUTXOID, options...)
}

// GetAtomicUTXOs returns the byte representation of the atomic UTXOs controlled
// by addrs from sourceChain.
func (c *Client) GetAtomicUTXOs(
	ctx context.Context,
	addrs []ids.ShortID,
	sourceChain string,
	limit uint32,
	startAddress ids.ShortID,
	startUTXOID ids.ID,
	options ...rpc.Option,
) ([][]byte, ids.ShortID, ids.ID, error) {
	// Format addresses in bech32 format (P-lux1..., P-local1..., etc.)
	formattedAddrs, err := c.formatAddresses(addrs)
	if err != nil {
		return nil, ids.ShortID{}, ids.Empty, err
	}

	// Build start index - only include address/UTXO if they're non-empty
	var startIndex api.Index
	if startAddress != ids.ShortEmpty || startUTXOID != ids.Empty {
		hrp := constants.GetHRP(c.networkID)
		startAddrStr, err := address.Format("P", hrp, startAddress[:])
		if err != nil {
			return nil, ids.ShortID{}, ids.Empty, err
		}
		startIndex = api.Index{
			Address: startAddrStr,
			UTXO:    startUTXOID.String(),
		}
	}

	res := &api.GetUTXOsReply{}
	err = c.Requester.SendRequest(ctx, "platform.getUTXOs", &api.GetUTXOsArgs{
		Addresses:   formattedAddrs,
		SourceChain: sourceChain,
		Limit:       json.Uint32(limit),
		StartIndex:  startIndex,
		Encoding:    formatting.Hex,
	}, res, options...)
	if err != nil {
		return nil, ids.ShortID{}, ids.Empty, err
	}

	utxos := make([][]byte, len(res.UTXOs))
	for i, utxo := range res.UTXOs {
		utxoBytes, err := formatting.Decode(res.Encoding, utxo)
		if err != nil {
			return nil, ids.ShortID{}, ids.Empty, err
		}
		utxos[i] = utxoBytes
	}
	endAddr, err := address.ParseToID(res.EndIndex.Address)
	if err != nil {
		return nil, ids.ShortID{}, ids.Empty, err
	}
	endUTXOID, err := ids.FromString(res.EndIndex.UTXO)
	return utxos, endAddr, endUTXOID, err
}

// GetNetClientResponse is the response from calling GetNet on the client
type GetNetClientResponse struct {
	// whether it is permissioned or not
	IsPermissioned bool
	// net auth information for a permissioned chain
	ControlKeys []ids.ShortID
	Threshold   uint32
	Locktime    uint64
	// net transformation tx ID for a permissionless chain
	NetTransformationTxID ids.ID
	// chain conversion information for an L1
	ConversionID   ids.ID
	ManagerChainID ids.ID
	ManagerAddress []byte
}

// GetNet returns information about the specified chain.
func (c *Client) GetNet(ctx context.Context, chainID ids.ID, options ...rpc.Option) (GetNetClientResponse, error) {
	res := &GetNetResponse{}
	err := c.Requester.SendRequest(ctx, "platform.getNet", &GetNetArgs{
		ChainID: chainID,
	}, res, options...)
	if err != nil {
		return GetNetClientResponse{}, err
	}
	controlKeys, err := address.ParseToIDs(res.ControlKeys)
	if err != nil {
		return GetNetClientResponse{}, err
	}

	return GetNetClientResponse{
		IsPermissioned:        res.IsPermissioned,
		ControlKeys:           controlKeys,
		Threshold:             uint32(res.Threshold),
		Locktime:              uint64(res.Locktime),
		NetTransformationTxID: res.NetTransformationTxID,
		ConversionID:          res.ConversionID,
		ManagerChainID:        res.ManagerChainID,
		ManagerAddress:        res.ManagerAddress,
	}, nil
}

// ClientNet is a representation of a net used in client methods
type ClientNet struct {
	// ID of the chain
	ID ids.ID
	// Each element of [ControlKeys] the address of a public key.
	// A transaction to add a validator to this net requires
	// signatures from [Threshold] of these keys to be valid.
	ControlKeys []ids.ShortID
	Threshold   uint32
}

// GetNets returns information about the specified chains
//
// Deprecated: Nets should be fetched from a dedicated indexer.
func (c *Client) GetNets(ctx context.Context, ids []ids.ID, options ...rpc.Option) ([]ClientNet, error) {
	res := &GetNetsResponse{}
	err := c.Requester.SendRequest(ctx, "platform.getNets", &GetNetsArgs{
		IDs: ids,
	}, res, options...)
	if err != nil {
		return nil, err
	}
	chains := make([]ClientNet, len(res.Nets))
	for i, apiNet := range res.Nets {
		controlKeys, err := address.ParseToIDs(apiNet.ControlKeys)
		if err != nil {
			return nil, err
		}

		chains[i] = ClientNet{
			ID:          apiNet.ID,
			ControlKeys: controlKeys,
			Threshold:   uint32(apiNet.Threshold),
		}
	}
	return chains, nil
}

// GetStakingAssetID returns the assetID of the asset used for staking on the
// chain corresponding to chainID.
func (c *Client) GetStakingAssetID(ctx context.Context, chainID ids.ID, options ...rpc.Option) (ids.ID, error) {
	res := &GetStakingAssetIDResponse{}
	err := c.Requester.SendRequest(ctx, "platform.getStakingAssetID", &GetStakingAssetIDArgs{
		ChainID: chainID,
	}, res, options...)
	return res.AssetID, err
}

// GetCurrentValidators returns the list of current validators for chainID.
func (c *Client) GetCurrentValidators(
	ctx context.Context,
	chainID ids.ID,
	nodeIDs []ids.NodeID,
	options ...rpc.Option,
) ([]ClientPermissionlessValidator, error) {
	res := &GetCurrentValidatorsReply{}
	err := c.Requester.SendRequest(ctx, "platform.getCurrentValidators", &GetCurrentValidatorsArgs{
		ChainID: chainID,
		NodeIDs: nodeIDs,
	}, res, options...)
	if err != nil {
		return nil, err
	}
	return getClientPermissionlessValidators(res.Validators)
}

// L1Validator is the response from calling GetL1Validator on the API client.
type L1Validator struct {
	ChainID               ids.ID
	NodeID                ids.NodeID
	PublicKey             *bls.PublicKey
	RemainingBalanceOwner *secp256k1fx.OutputOwners
	DeactivationOwner     *secp256k1fx.OutputOwners
	StartTime             uint64
	Weight                uint64
	MinNonce              uint64
	// Balance is the remaining amount of LUX this L1 validator has for paying
	// the continuous fee.
	Balance uint64
}

// GetL1Validator returns the requested L1 validator with validationID and the
// height at which it was calculated.
func (c *Client) GetL1Validator(
	ctx context.Context,
	validationID ids.ID,
	options ...rpc.Option,
) (L1Validator, uint64, error) {
	res := &GetL1ValidatorReply{}
	err := c.Requester.SendRequest(ctx, "platform.getL1Validator",
		&GetL1ValidatorArgs{
			ValidationID: validationID,
		},
		res, options...,
	)
	if err != nil {
		return L1Validator{}, 0, err
	}
	var pk *bls.PublicKey
	if res.PublicKey != nil {
		pk, err = bls.PublicKeyFromCompressedBytes(*res.PublicKey)
		if err != nil {
			return L1Validator{}, 0, err
		}
	}
	remainingBalanceOwnerAddrs, err := address.ParseToIDs(res.RemainingBalanceOwner.Addresses)
	if err != nil {
		return L1Validator{}, 0, err
	}
	deactivationOwnerAddrs, err := address.ParseToIDs(res.DeactivationOwner.Addresses)
	if err != nil {
		return L1Validator{}, 0, err
	}

	var minNonce uint64
	if res.MinNonce != nil {
		minNonce = uint64(*res.MinNonce)
	}
	var balance uint64
	if res.Balance != nil {
		balance = uint64(*res.Balance)
	}

	return L1Validator{
		ChainID:   res.ChainID,
		NodeID:    res.NodeID,
		PublicKey: pk,
		RemainingBalanceOwner: &secp256k1fx.OutputOwners{
			Locktime:  uint64(res.RemainingBalanceOwner.Locktime),
			Threshold: uint32(res.RemainingBalanceOwner.Threshold),
			Addrs:     remainingBalanceOwnerAddrs,
		},
		DeactivationOwner: &secp256k1fx.OutputOwners{
			Locktime:  uint64(res.DeactivationOwner.Locktime),
			Threshold: uint32(res.DeactivationOwner.Threshold),
			Addrs:     deactivationOwnerAddrs,
		},
		StartTime: uint64(res.StartTime),
		Weight:    uint64(res.Weight),
		MinNonce:  minNonce,
		Balance:   balance,
	}, uint64(res.Height), err
}

// GetCurrentSupply returns an upper bound on the supply of LUX in the system
// along with the chain height.
func (c *Client) GetCurrentSupply(ctx context.Context, chainID ids.ID, options ...rpc.Option) (uint64, uint64, error) {
	res := &GetCurrentSupplyReply{}
	err := c.Requester.SendRequest(ctx, "platform.getCurrentSupply", &GetCurrentSupplyArgs{
		ChainID: chainID,
	}, res, options...)
	return uint64(res.Supply), uint64(res.Height), err
}

// SampleValidators returns the nodeIDs of a sample of sampleSize validators
// from the current validator set for chainID.
func (c *Client) SampleValidators(ctx context.Context, chainID ids.ID, sampleSize uint16, options ...rpc.Option) ([]ids.NodeID, error) {
	res := &SampleValidatorsReply{}
	err := c.Requester.SendRequest(ctx, "platform.sampleValidators", &SampleValidatorsArgs{
		ChainID: chainID,
		Size:    json.Uint16(sampleSize),
	}, res, options...)
	return res.Validators, err
}

// GetBlockchainStatus returns the current status of blockchainID.
func (c *Client) GetBlockchainStatus(ctx context.Context, blockchainID string, options ...rpc.Option) (status.BlockchainStatus, error) {
	res := &GetBlockchainStatusReply{}
	err := c.Requester.SendRequest(ctx, "platform.getBlockchainStatus", &GetBlockchainStatusArgs{
		BlockchainID: blockchainID,
	}, res, options...)
	return res.Status, err
}

// ValidatedBy returns the chainID that validates blockchainID.
func (c *Client) ValidatedBy(ctx context.Context, blockchainID ids.ID, options ...rpc.Option) (ids.ID, error) {
	res := &ValidatedByResponse{}
	err := c.Requester.SendRequest(ctx, "platform.validatedBy", &ValidatedByArgs{
		BlockchainID: blockchainID,
	}, res, options...)
	return res.ChainID, err
}

// Validates returns the list of blockchains that are validated by chainID.
func (c *Client) Validates(ctx context.Context, chainID ids.ID, options ...rpc.Option) ([]ids.ID, error) {
	res := &ValidatesResponse{}
	err := c.Requester.SendRequest(ctx, "platform.validates", &ValidatesArgs{
		ChainID: chainID,
	}, res, options...)
	return res.BlockchainIDs, err
}

// GetBlockchains returns the list of all blockchains on the platform.
//
// Deprecated: Blockchains should be fetched from a dedicated indexer.
func (c *Client) GetBlockchains(ctx context.Context, options ...rpc.Option) ([]APIBlockchain, error) {
	res := &GetBlockchainsResponse{}
	err := c.Requester.SendRequest(ctx, "platform.getBlockchains", struct{}{}, res, options...)
	return res.Blockchains, err
}

// IssueTx issues the transaction and returns its txID.
func (c *Client) IssueTx(ctx context.Context, txBytes []byte, options ...rpc.Option) (ids.ID, error) {
	txStr, err := formatting.Encode(formatting.Hex, txBytes)
	if err != nil {
		return ids.Empty, err
	}

	res := &api.JSONTxID{}
	err = c.Requester.SendRequest(ctx, "platform.issueTx", &api.FormattedTx{
		Tx:       txStr,
		Encoding: formatting.Hex,
	}, res, options...)
	return res.TxID, err
}

// GetTx returns the byte representation of txID.
func (c *Client) GetTx(ctx context.Context, txID ids.ID, options ...rpc.Option) ([]byte, error) {
	res := &api.FormattedTx{}
	err := c.Requester.SendRequest(ctx, "platform.getTx", &api.GetTxArgs{
		TxID:     txID,
		Encoding: formatting.Hex,
	}, res, options...)
	if err != nil {
		return nil, err
	}
	return formatting.Decode(res.Encoding, res.Tx)
}

// GetTxStatus returns the status of txID.
func (c *Client) GetTxStatus(ctx context.Context, txID ids.ID, options ...rpc.Option) (*GetTxStatusResponse, error) {
	res := &GetTxStatusResponse{}
	err := c.Requester.SendRequest(
		ctx,
		"platform.getTxStatus",
		&GetTxStatusArgs{
			TxID: txID,
		},
		res,
		options...,
	)
	return res, err
}

// GetStake returns the amount of nLUX that addrs have cumulatively staked on
// the Primary Network.
//
// Deprecated: Stake should be calculated using GetTx and GetCurrentValidators.
func (c *Client) GetStake(
	ctx context.Context,
	addrs []ids.ShortID,
	validatorsOnly bool,
	options ...rpc.Option,
) (map[ids.ID]uint64, [][]byte, error) {
	res := &GetStakeReply{}
	err := c.Requester.SendRequest(ctx, "platform.getStake", &GetStakeArgs{
		JSONAddresses: api.JSONAddresses{
			Addresses: ids.ShortIDsToStrings(addrs),
		},
		ValidatorsOnly: validatorsOnly,
		Encoding:       formatting.Hex,
	}, res, options...)
	if err != nil {
		return nil, nil, err
	}

	staked := make(map[ids.ID]uint64, len(res.Stakeds))
	for assetID, amount := range res.Stakeds {
		staked[assetID] = uint64(amount)
	}

	outputs := make([][]byte, len(res.Outputs))
	for i, outputStr := range res.Outputs {
		output, err := formatting.Decode(res.Encoding, outputStr)
		if err != nil {
			return nil, nil, err
		}
		outputs[i] = output
	}
	return staked, outputs, err
}

// GetMinStake returns the minimum staking amount in nLUX for validators and
// delegators respectively.
func (c *Client) GetMinStake(ctx context.Context, chainID ids.ID, options ...rpc.Option) (uint64, uint64, error) {
	res := &GetMinStakeReply{}
	err := c.Requester.SendRequest(ctx, "platform.getMinStake", &GetMinStakeArgs{
		ChainID: chainID,
	}, res, options...)
	return uint64(res.MinValidatorStake), uint64(res.MinDelegatorStake), err
}

// GetTotalStake returns the total amount (in nLUX) staked on the network.
func (c *Client) GetTotalStake(ctx context.Context, chainID ids.ID, options ...rpc.Option) (uint64, error) {
	res := &GetTotalStakeReply{}
	err := c.Requester.SendRequest(ctx, "platform.getTotalStake", &GetTotalStakeArgs{
		ChainID: chainID,
	}, res, options...)
	var amount json.Uint64
	if chainID == constants.PrimaryNetworkID {
		amount = res.Stake
	} else {
		amount = res.Weight
	}
	return uint64(amount), err
}

// GetRewardUTXOs returns the reward UTXOs for a transaction.
//
// Deprecated: GetRewardUTXOs should be fetched from a dedicated indexer.
func (c *Client) GetRewardUTXOs(ctx context.Context, args *api.GetTxArgs, options ...rpc.Option) ([][]byte, error) {
	res := &GetRewardUTXOsReply{}
	err := c.Requester.SendRequest(ctx, "platform.getRewardUTXOs", args, res, options...)
	if err != nil {
		return nil, err
	}
	utxos := make([][]byte, len(res.UTXOs))
	for i, utxoStr := range res.UTXOs {
		utxoBytes, err := formatting.Decode(res.Encoding, utxoStr)
		if err != nil {
			return nil, err
		}
		utxos[i] = utxoBytes
	}
	return utxos, err
}

// GetTimestamp returns the current chain timestamp.
func (c *Client) GetTimestamp(ctx context.Context, options ...rpc.Option) (time.Time, error) {
	res := &GetTimestampReply{}
	err := c.Requester.SendRequest(ctx, "platform.getTimestamp", struct{}{}, res, options...)
	return res.Timestamp, err
}

// GetValidatorsAt returns the weights of the validator set of a provided chain
// at the specified height or at proposerVM height if set to
// [platformapi.ProposedHeight].
func (c *Client) GetValidatorsAt(
	ctx context.Context,
	chainID ids.ID,
	height platformapi.Height,
	options ...rpc.Option,
) (map[ids.NodeID]*validators.GetValidatorOutput, error) {
	res := &GetValidatorsAtReply{}
	err := c.Requester.SendRequest(ctx, "platform.getValidatorsAt", &GetValidatorsAtArgs{
		ChainID: chainID,
		Height:  height,
	}, res, options...)
	return res.Validators, err
}

// GetBlock returns blockID.
func (c *Client) GetBlock(ctx context.Context, blockID ids.ID, options ...rpc.Option) ([]byte, error) {
	res := &api.FormattedBlock{}
	if err := c.Requester.SendRequest(ctx, "platform.getBlock", &api.GetBlockArgs{
		BlockID:  blockID,
		Encoding: formatting.Hex,
	}, res, options...); err != nil {
		return nil, err
	}
	return formatting.Decode(res.Encoding, res.Block)
}

// GetBlockByHeight returns the block at the given height.
func (c *Client) GetBlockByHeight(ctx context.Context, height uint64, options ...rpc.Option) ([]byte, error) {
	res := &api.FormattedBlock{}
	err := c.Requester.SendRequest(ctx, "platform.getBlockByHeight", &api.GetBlockByHeightArgs{
		Height:   json.Uint64(height),
		Encoding: formatting.HexNC,
	}, res, options...)
	if err != nil {
		return nil, err
	}
	return formatting.Decode(res.Encoding, res.Block)
}

// GetFeeConfig returns the dynamic fee config.
func (c *Client) GetFeeConfig(ctx context.Context, options ...rpc.Option) (*gas.Config, error) {
	res := &gas.Config{}
	err := c.Requester.SendRequest(ctx, "platform.getFeeConfig", struct{}{}, res, options...)
	return res, err
}

// GetFeeState returns the current fee state.
func (c *Client) GetFeeState(ctx context.Context, options ...rpc.Option) (
	gas.State,
	gas.Price,
	time.Time,
	error,
) {
	res := &GetFeeStateReply{}
	err := c.Requester.SendRequest(ctx, "platform.getFeeState", struct{}{}, res, options...)
	return res.State, res.Price, res.Time, err
}

// GetValidatorFeeConfig returns the validator fee config.
func (c *Client) GetValidatorFeeConfig(ctx context.Context, options ...rpc.Option) (*fee.Config, error) {
	res := &fee.Config{}
	err := c.Requester.SendRequest(ctx, "platform.getValidatorFeeConfig", struct{}{}, res, options...)
	return res, err
}

// GetValidatorFeeState returns the current validator fee state.
func (c *Client) GetValidatorFeeState(ctx context.Context, options ...rpc.Option) (
	gas.Gas,
	gas.Price,
	time.Time,
	error,
) {
	res := &GetValidatorFeeStateReply{}
	err := c.Requester.SendRequest(ctx, "platform.getValidatorFeeState", struct{}{}, res, options...)
	return res.Excess, res.Price, res.Time, err
}

func AwaitTxAccepted(
	c *Client,
	ctx context.Context,
	txID ids.ID,
	freq time.Duration,
	options ...rpc.Option,
) error {
	ticker := time.NewTicker(freq)
	defer ticker.Stop()

	for {
		res, err := c.GetTxStatus(ctx, txID, options...)
		if err != nil {
			return err
		}

		switch res.Status {
		case status.Committed, status.Aborted:
			return nil
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// GetNetOwners returns a map of chain ID to current chain's owner
func GetNetOwners(
	c *Client,
	ctx context.Context,
	chainIDs ...ids.ID,
) (map[ids.ID]fx.Owner, error) {
	chainOwners := make(map[ids.ID]fx.Owner, len(chainIDs))
	for _, chainID := range chainIDs {
		chainInfo, err := c.GetNet(ctx, chainID)
		if err != nil {
			return nil, err
		}
		chainOwners[chainID] = &secp256k1fx.OutputOwners{
			Locktime:  chainInfo.Locktime,
			Threshold: chainInfo.Threshold,
			Addrs:     chainInfo.ControlKeys,
		}
	}
	return chainOwners, nil
}

// GetDeactivationOwners returns a map of validation ID to deactivation owners
func GetDeactivationOwners(
	c *Client,
	ctx context.Context,
	validationIDs ...ids.ID,
) (map[ids.ID]fx.Owner, error) {
	deactivationOwners := make(map[ids.ID]fx.Owner, len(validationIDs))
	for _, validationID := range validationIDs {
		l1Validator, _, err := c.GetL1Validator(ctx, validationID)
		if err != nil {
			return nil, err
		}
		deactivationOwners[validationID] = l1Validator.DeactivationOwner
	}
	return deactivationOwners, nil
}

// GetOwners returns the union of GetNetOwners and GetDeactivationOwners.
func GetOwners(
	c *Client,
	ctx context.Context,
	chainIDs []ids.ID,
	validationIDs []ids.ID,
) (map[ids.ID]fx.Owner, error) {
	chainOwners, err := GetNetOwners(c, ctx, chainIDs...)
	if err != nil {
		return nil, err
	}
	deactivationOwners, err := GetDeactivationOwners(c, ctx, validationIDs...)
	if err != nil {
		return nil, err
	}

	owners := make(map[ids.ID]fx.Owner, len(chainOwners)+len(deactivationOwners))
	for id, owner := range chainOwners {
		owners[id] = owner
	}
	for id, owner := range deactivationOwners {
		owners[id] = owner
	}
	return owners, nil
}
