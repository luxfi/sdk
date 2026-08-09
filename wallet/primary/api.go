// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

// Package primary provides primary network wallet operations.
package primary

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/luxfi/geth/ethclient"

	"github.com/luxfi/constants"
	"github.com/luxfi/formatting"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/rpc"
	sdkinfo "github.com/luxfi/sdk/info"
	"github.com/luxfi/sdk/platformvm"
	"github.com/luxfi/sdk/wallet/chain/c"
	"github.com/luxfi/sdk/wallet/chain/p"
	"github.com/luxfi/sdk/wallet/chain/x"

	gethcommon "github.com/luxfi/geth/common"
	pbuilder "github.com/luxfi/sdk/wallet/chain/p/builder"
	xbuilder "github.com/luxfi/sdk/wallet/chain/x/builder"
	walletcommon "github.com/luxfi/sdk/wallet/primary/common"
)

const (
	MainnetAPIURI = "https://api.lux.network"
	TestnetAPIURI = "https://api.lux-test.network"
	LocalAPIURI   = "http://localhost:9630"

	fetchLimit = 1024

	// Retry configuration for transient network errors
	maxRetries    = 5
	retryBaseWait = 100 * time.Millisecond
)

// perform their own assertions.
var (
	_ UTXOClient = (*platformvm.Client)(nil)
	_ UTXOClient = (*XClient)(nil)
)

type UTXOClient interface {
	GetAtomicUTXOs(
		ctx context.Context,
		addrs []ids.ShortID,
		sourceChain string,
		limit uint32,
		startAddress ids.ShortID,
		startUTXOID ids.ID,
		options ...rpc.Option,
	) ([][]byte, ids.ShortID, ids.ID, error)
}

// XClient is a client for interacting with the X-Chain
type XClient struct {
	requester    rpc.EndpointRequester
	networkID    uint32
	blockchainID ids.ID
}

// NewXClient returns a new X-Chain client
func NewXClient(uri, chainAlias string) *XClient {
	return &XClient{
		requester: rpc.NewEndpointRequester(
			fmt.Sprintf("%s/v1/bc/%s", uri, chainAlias),
		),
	}
}

// NewXClientWithContext returns a new X-Chain client with context information
// required for proper address formatting and UTXO queries.
// Note: Uses "X" alias for the endpoint (more reliable) but keeps blockchainID
// for address formatting in UTXO queries.
func NewXClientWithContext(uri string, networkID uint32, blockchainID ids.ID) *XClient {
	return &XClient{
		requester: rpc.NewEndpointRequester(
			fmt.Sprintf("%s/v1/bc/X", uri), // Use alias, not blockchain ID (avoids EOF issues)
		),
		networkID:    networkID,
		blockchainID: blockchainID,
	}
}

// SetContext sets the network context for address formatting
func (c *XClient) SetContext(networkID uint32, blockchainID ids.ID) {
	c.networkID = networkID
	c.blockchainID = blockchainID
}

// GetAtomicUTXOs implements UTXOClient.
// Queries the X-chain for UTXOs controlled by the given addresses.
func (c *XClient) GetAtomicUTXOs(
	ctx context.Context,
	addrs []ids.ShortID,
	sourceChain string,
	limit uint32,
	startAddress ids.ShortID,
	startUTXOID ids.ID,
	options ...rpc.Option,
) ([][]byte, ids.ShortID, ids.ID, error) {
	// Format addresses using blockchain ID prefix for local networks
	formattedAddrs := make([]string, len(addrs))
	hrp := constants.GetHRP(c.networkID)
	for i, addr := range addrs {
		// Use blockchain ID as chain prefix for proper address formatting
		chainPrefix := c.blockchainID.String()
		if chainPrefix == "" || c.blockchainID == ids.Empty {
			chainPrefix = "X" // fallback to "X" for compatibility
		}
		addrStr, err := formatAddressWithChain(chainPrefix, hrp, addr[:])
		if err != nil {
			return nil, ids.ShortID{}, ids.Empty, fmt.Errorf("failed to format address: %w", err)
		}
		formattedAddrs[i] = addrStr
	}

	res := &getUTXOsReply{}
	err := c.requester.SendRequest(ctx, "xvm.getUTXOs", &getUTXOsArgs{
		Addresses:   formattedAddrs,
		SourceChain: sourceChain,
		Limit:       limit,
		Encoding:    "hex",
	}, res, options...)
	if err != nil {
		return nil, ids.ShortID{}, ids.Empty, fmt.Errorf("failed to get UTXOs: %w", err)
	}

	utxos := make([][]byte, len(res.UTXOs))
	for i, utxo := range res.UTXOs {
		utxoBytes, err := formatting.Decode(formatting.Hex, utxo)
		if err != nil {
			return nil, ids.ShortID{}, ids.Empty, fmt.Errorf("failed to decode UTXO %d: %w", i, err)
		}
		utxos[i] = utxoBytes
	}

	// Parse end index for pagination
	var endAddr ids.ShortID
	var endUTXO ids.ID
	if res.EndIndex.Address != "" {
		_, _, addrBytes, err := parseAddress(res.EndIndex.Address)
		if err == nil && len(addrBytes) == 20 {
			copy(endAddr[:], addrBytes)
		}
	}
	if res.EndIndex.UTXO != "" {
		endUTXO, _ = ids.FromString(res.EndIndex.UTXO)
	}

	return utxos, endAddr, endUTXO, nil
}

// getUTXOsArgs are the arguments to xvm.getUTXOs
type getUTXOsArgs struct {
	Addresses   []string `json:"addresses"`
	SourceChain string   `json:"sourceChain,omitempty"`
	Limit       uint32   `json:"limit,omitempty"`
	StartIndex  struct {
		Address string `json:"address,omitempty"`
		UTXO    string `json:"utxo,omitempty"`
	} `json:"startIndex,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

// getUTXOsReply is the response from xvm.getUTXOs
type getUTXOsReply struct {
	NumFetched string   `json:"numFetched"`
	UTXOs      []string `json:"utxos"`
	EndIndex   struct {
		Address string `json:"address"`
		UTXO    string `json:"utxo"`
	} `json:"endIndex"`
	Encoding string `json:"encoding"`
}

type LUXState struct {
	PClient *platformvm.Client
	PCTX    *pbuilder.Context
	// X-Chain: opt-in via AttachXChain. Nil if not attached or not registered on this network.
	XClient *XClient
	XCTX    *xbuilder.Context
	// C-Chain: opt-in via AttachCChain. EVM, fundamentally different from
	// UTXO P/X — uses ethclient. Nil if not attached or not registered.
	// CCTX would live in ./chain/c (EthClient + ChainID + GasPrice) — not
	// a UTXOCtx. Wired through FetchEthState today.
	UTXOs walletcommon.UTXOs

	// uri and addrs are preserved so AttachXChain / AttachCChain can reuse them.
	uri   string
	addrs []ids.ShortID
}

// FetchPState fetches ONLY the P-Chain client + context + UTXOs.
//
// This is the canonical entry point. P-Chain is the only required chain
// for sovereign-L1 spawn (CreateChainTx), validator ops, primary network
// transactions. X-Chain (UTXO asset transfers) and C-Chain (EVM smart
// contracts) are opt-in — call AttachXChain / AttachCChain on the
// returned state if you need them.
//
// Sovereign-L1 callers: this is the function you want.
// Don't call FetchState (it tries to pull X + C, which is wasteful when
// they're not needed and breaks against P-only Quasar networks if not
// fail-softed correctly).
func FetchPState(
	ctx context.Context,
	uri string,
	addrs set.Set[ids.ShortID],
) (*LUXState, error) {
	infoClient := sdkinfo.NewClient(uri)
	pClient := platformvm.NewClient(uri)

	pCTX, err := p.NewContextFromClients(ctx, infoClient, pClient)
	if err != nil {
		return nil, err
	}
	// Set the network ID on the pClient for proper bech32 address formatting.
	// Without this, the client uses networkID=0 which maps to "custom" HRP fallback.
	pClient.SetNetworkID(pCTX.NetworkID)

	utxos := walletcommon.NewUTXOs()
	if err := AddAllUTXOs(
		ctx,
		utxos,
		pClient,
		constants.PlatformChainID,
		constants.PlatformChainID,
		addrs.List(),
	); err != nil {
		return nil, err
	}

	return &LUXState{
		PClient: pClient,
		PCTX:    pCTX,
		UTXOs:   utxos,
		uri:     uri,
		addrs:   addrs.List(),
	}, nil
}

// AttachXChain adds X-Chain client + context to the state. Fail-soft:
// if X-Chain is not registered on this network (Quasar mainnet is P+C
// only), returns the state unchanged with no error. Any other RPC
// error is returned.
func (s *LUXState) AttachXChain(ctx context.Context) error {
	if s.XCTX != nil {
		return nil // idempotent
	}
	infoClient := sdkinfo.NewClient(s.uri)
	utxoAssetID := s.PCTX.UTXOAssetID
	const (
		baseTxFee        = uint64(1000000)  // 0.001 LUX
		createAssetTxFee = uint64(10000000) // 0.01 LUX
	)
	xCTX, err := x.NewContextFromClients(ctx, infoClient, utxoAssetID, baseTxFee, createAssetTxFee)
	if err != nil {
		if isXChainNotEnabled(err) {
			return nil // X-Chain not registered — opt-in attach is a no-op
		}
		return err
	}
	s.XCTX = xCTX
	s.XClient = NewXClientWithContext(s.uri, s.PCTX.NetworkID, xCTX.BlockchainID)
	// Without this the X wallet is built over an empty set and every tx
	// reports insufficient funds regardless of balance.
	return AddAllUTXOs(
		ctx,
		s.UTXOs,
		s.XClient,
		xCTX.BlockchainID,
		xCTX.BlockchainID,
		s.addrs,
	)
}

// FetchState is the P+X convenience that pre-fetches P (required) and
// X (opt-in via AttachXChain). Kept for back-compat. New callers
// should use FetchPState + AttachXChain / AttachCChain explicitly so
// the dependency graph is visible at the call site.
func FetchState(
	ctx context.Context,
	uri string,
	addrs set.Set[ids.ShortID],
) (*LUXState, error) {
	state, err := FetchPState(ctx, uri, addrs)
	if err != nil {
		return nil, err
	}
	if err := state.AttachXChain(ctx); err != nil {
		return nil, err
	}
	return state, nil
}

type EthState struct {
	Client   *ethclient.Client
	Accounts map[gethcommon.Address]*c.Account
}

func FetchEthState(
	ctx context.Context,
	uri string,
	addrs set.Set[gethcommon.Address],
) (*EthState, error) {
	path := fmt.Sprintf(
		"%s/v1/%s/C/rpc",
		uri,
		constants.ChainAliasPrefix,
	)
	client, err := ethclient.Dial(path)
	if err != nil {
		return nil, err
	}

	accounts := make(map[gethcommon.Address]*c.Account, addrs.Len())
	for addr := range addrs {
		// Convert ethereum address to geth address
		gethAddr := gethcommon.Address(addr)
		balance, err := client.BalanceAt(ctx, gethAddr, nil)
		if err != nil {
			return nil, err
		}
		nonce, err := client.NonceAt(ctx, gethAddr, nil)
		if err != nil {
			return nil, err
		}
		accounts[addr] = &c.Account{
			Balance: balance,
			Nonce:   nonce,
		}
	}
	return &EthState{
		Client:   client,
		Accounts: accounts,
	}, nil
}

// isRetryableError checks if an error is transient and should be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for EOF errors (connection closed by server)
	if err == io.EOF || strings.Contains(errStr, "EOF") {
		return true
	}
	// Check for connection reset
	if strings.Contains(errStr, "connection reset") {
		return true
	}
	// Check for temporary network errors
	if strings.Contains(errStr, "temporary failure") {
		return true
	}
	return false
}

// AddAllUTXOs fetches all the UTXOs referenced by [addresses] that were sent
// from [sourceChainID] to [destinationChainID] from the [client] and adds them
// into [utxos]. If [ctx] expires, then the returned error will be immediately
// reported.
//
// UTXOs are read ZAP-native (see zapUTXO): the node serves the envelope
// utxo.UTXO.WireBytes() produces, and the zero-copy wire accessors read it
// directly. No codec.Manager and no intermediate encoding sit on this path.
func AddAllUTXOs(
	ctx context.Context,
	utxos walletcommon.UTXOs,
	client UTXOClient,
	sourceChainID ids.ID,
	destinationChainID ids.ID,
	addrs []ids.ShortID,
) error {
	// When sourceChainID == destinationChainID, we're fetching regular UTXOs
	// on the same chain, not atomic UTXOs from cross-chain transfers.
	// The platformvm service expects empty string for regular UTXO fetches,
	// because at runtime the VM's chainID may differ from the constant.
	var sourceChainIDStr string
	if sourceChainID != destinationChainID {
		sourceChainIDStr = sourceChainID.String()
	}
	var (
		startAddr ids.ShortID
		startUTXO ids.ID
	)
	for {
		var utxosBytes [][]byte
		var endAddr ids.ShortID
		var endUTXO ids.ID
		var err error

		// Retry loop for transient network errors
		for retry := 0; retry <= maxRetries; retry++ {
			utxosBytes, endAddr, endUTXO, err = client.GetAtomicUTXOs(
				ctx,
				addrs,
				sourceChainIDStr,
				fetchLimit,
				startAddr,
				startUTXO,
			)
			if err == nil {
				break
			}
			if !isRetryableError(err) {
				return fmt.Errorf("GetAtomicUTXOs: %w", err)
			}
			if retry == maxRetries {
				return fmt.Errorf("GetAtomicUTXOs failed after %d retries: %w", maxRetries, err)
			}
			// Wait with exponential backoff before retrying
			wait := retryBaseWait * time.Duration(1<<uint(retry))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
				// Continue to next retry
			}
		}

		for _, utxoBytes := range utxosBytes {
			utxo, err := zapUTXO(utxoBytes)
			if err != nil {
				// A UTXO the node served and we cannot read is a wire
				// disagreement, not a spendability question. Report it —
				// silently skipping yields an empty set and surfaces later
				// as an inscrutable "insufficient funds".
				return fmt.Errorf("unreadable UTXO from %s: %w", sourceChainID, err)
			}
			if utxo.AssetID() == ids.Empty {
				continue // invalid UTXO
			}
			if err := utxos.AddUTXO(ctx, sourceChainID, destinationChainID, utxo); err != nil {
				return err
			}
		}

		if len(utxosBytes) < fetchLimit {
			break
		}

		// Update the vars to query the next page of UTXOs.
		startAddr = endAddr
		startUTXO = endUTXO
	}
	return nil
}

// formatAddressWithChain formats an address with chain prefix and bech32 encoding
func formatAddressWithChain(chainPrefix, hrp string, addrBytes []byte) (string, error) {
	// Convert to bech32 with 5-bit groups
	conv, err := bech32.ConvertBits(addrBytes, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("failed to convert address bits: %w", err)
	}
	encoded, err := bech32.Encode(hrp, conv)
	if err != nil {
		return "", fmt.Errorf("failed to bech32 encode: %w", err)
	}
	return fmt.Sprintf("%s-%s", chainPrefix, encoded), nil
}

// parseAddress parses a formatted address like "X-lux1..." into its components
func parseAddress(addr string) (chainPrefix, hrp string, addrBytes []byte, err error) {
	// Split on "-" to get chain prefix
	parts := make([]string, 0, 2)
	idx := 0
	for i, c := range addr {
		if c == '-' {
			parts = append(parts, addr[idx:i])
			idx = i + 1
			if len(parts) == 1 {
				break
			}
		}
	}
	if idx < len(addr) {
		parts = append(parts, addr[idx:])
	}

	if len(parts) != 2 {
		return "", "", nil, fmt.Errorf("invalid address format: %s", addr)
	}
	chainPrefix = parts[0]
	bech32Addr := parts[1]

	// Decode bech32
	hrp, data, err := bech32.Decode(bech32Addr)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to decode bech32: %w", err)
	}

	// Convert from 5-bit to 8-bit groups
	addrBytes, err = bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to convert address bits: %w", err)
	}

	return chainPrefix, hrp, addrBytes, nil
}


// isXChainNotEnabled detects the canonical "info.getBlockchainID returns
// no-such-alias for X" error that surfaces when running against a P-only
// network (one whose platform genesis does not include an XVM chain —
// e.g. Quasar-era mainnet which only registers P + C). We match by
// substring of the JSON-RPC error message rather than a typed sentinel
// because the upstream info-service emits a free-form string; re-evaluate
// this matcher if/when info.getBlockchainID gains a typed error path.
//
// Kept in lockstep with node/wallet/network/primary/api.go:isXChainNotEnabled
// so the SDK and node wallets fail-soft on the same set of error messages.
func isXChainNotEnabled(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "there is no id with alias: x") ||
		strings.Contains(msg, "no chain with alias")
}
