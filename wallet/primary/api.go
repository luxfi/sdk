// Copyright (C) 2019-2025, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

// Package primary provides primary network wallet operations.
package primary

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/luxfi/geth/ethclient"

	"github.com/luxfi/codec"
	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	"github.com/luxfi/math/set"
	"github.com/luxfi/rpc"
	"github.com/luxfi/sdk/api/info"
	"github.com/luxfi/sdk/wallet/chain/c"
	"github.com/luxfi/sdk/wallet/chain/p"
	"github.com/luxfi/sdk/wallet/chain/x"
	"github.com/luxfi/vm/components/lux"
	"github.com/luxfi/vm/vms/platformvm"

	gethcommon "github.com/luxfi/geth/common"
	pbuilder "github.com/luxfi/sdk/wallet/chain/p/builder"
	xbuilder "github.com/luxfi/sdk/wallet/chain/x/builder"
	walletcommon "github.com/luxfi/sdk/wallet/primary/common"
	ptxs "github.com/luxfi/vm/vms/platformvm/txs"
)

const (
	MainnetAPIURI = "https://api.lux.network"
	TestnetAPIURI = "https://api.lux-test.network"
	LocalAPIURI   = "http://localhost:9630"

	fetchLimit = 1024
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
			fmt.Sprintf("%s/ext/bc/%s", uri, chainAlias),
		),
	}
}

// NewXClientWithContext returns a new X-Chain client with context information
// required for proper address formatting and UTXO queries
func NewXClientWithContext(uri string, networkID uint32, blockchainID ids.ID) *XClient {
	return &XClient{
		requester: rpc.NewEndpointRequester(
			fmt.Sprintf("%s/ext/bc/%s", uri, blockchainID.String()),
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

	// Query xvm.getUTXOs
	res := &getUTXOsReply{}
	err := c.requester.SendRequest(ctx, "xvm.getUTXOs", &getUTXOsArgs{
		Addresses:   formattedAddrs,
		SourceChain: sourceChain,
		Limit:       limit,
		Encoding:    "hex",
	}, res, options...)
	if err != nil {
		// If xvm service not available, try avm for compatibility
		err2 := c.requester.SendRequest(ctx, "avm.getUTXOs", &getUTXOsArgs{
			Addresses:   formattedAddrs,
			SourceChain: sourceChain,
			Limit:       limit,
			Encoding:    "hex",
		}, res, options...)
		if err2 != nil {
			return nil, ids.ShortID{}, ids.Empty, fmt.Errorf("failed to get UTXOs: xvm error: %w, avm error: %v", err, err2)
		}
	}

	// Decode UTXOs from hex
	utxos := make([][]byte, len(res.UTXOs))
	for i, utxo := range res.UTXOs {
		utxoBytes, err := hexDecode(utxo)
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
	XClient *XClient
	XCTX    *xbuilder.Context
	// CClient *ethclient.Client // Not yet implemented
	// CCTX    *c.Context         // Not yet implemented
	UTXOs walletcommon.UTXOs
}

func FetchState(
	ctx context.Context,
	uri string,
	addrs set.Set[ids.ShortID],
) (
	*LUXState,
	error,
) {
	infoClient := info.NewClient(uri)
	pClient := platformvm.NewClient(uri)
	// C-chain client disabled for now
	// cClient, err := ethclient.Dial(uri + "/ext/bc/C/rpc")
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to connect to C-chain: %w", err)
	// }

	pCTX, err := p.NewContextFromClients(ctx, infoClient, pClient)
	if err != nil {
		return nil, err
	}

	// Set the network ID on the pClient for proper bech32 address formatting
	// Without this, the client uses networkID=0 which maps to "custom" HRP fallback
	pClient.SetNetworkID(pCTX.NetworkID)

	// For X-chain, we need to get the LUX asset ID and fees
	// TODO: Get these from the xClient or infoClient
	luxAssetID := pCTX.XAssetID
	baseTxFee := uint64(1000000)         // 0.001 LUX
	createAssetTxFee := uint64(10000000) // 0.01 LUX

	xCTX, err := x.NewContextFromClients(ctx, infoClient, luxAssetID, baseTxFee, createAssetTxFee)
	if err != nil {
		return nil, err
	}

	// Create X-chain client with context for proper address formatting
	xClient := NewXClientWithContext(uri, pCTX.NetworkID, xCTX.BlockchainID)

	// C-chain context disabled for now
	// cCTX, err := c.NewContextFromClients(ctx, infoClient, pClient)
	// if err != nil {
	// 	return nil, err
	// }

	utxos := walletcommon.NewUTXOs()
	addrList := addrs.List()
	chains := []struct {
		id     ids.ID
		client UTXOClient
		codec  codec.Manager
	}{
		{
			id:     constants.PlatformChainID,
			client: pClient,
			codec:  ptxs.Codec,
		},
		{
			id:     xCTX.BlockchainID,
			client: xClient,
			codec:  codec.NewDefaultManager(),
		},
		// {
		// 	id:     cCTX.BlockchainID,
		// 	client: cClient,
		// 	codec:  evm.Codec,
		// },
	}
	for _, destinationChain := range chains {
		for _, sourceChain := range chains {
			err = AddAllUTXOs(
				ctx,
				utxos,
				destinationChain.client,
				destinationChain.codec,
				sourceChain.id,
				destinationChain.id,
				addrList,
			)
			if err != nil {
				return nil, err
			}
		}
	}
	return &LUXState{
		PClient: pClient,
		PCTX:    pCTX,
		XClient: xClient,
		XCTX:    xCTX,
		// CClient: cClient, // Disabled
		// CCTX:    cCTX,    // Disabled
		UTXOs: utxos,
	}, nil
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
		"%s/ext/%s/C/rpc",
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

// AddAllUTXOs fetches all the UTXOs referenced by [addresses] that were sent
// from [sourceChainID] to [destinationChainID] from the [client]. It then uses
// [codec] to parse the returned UTXOs and it adds them into [utxos]. If [ctx]
// expires, then the returned error will be immediately reported.
func AddAllUTXOs(
	ctx context.Context,
	utxos walletcommon.UTXOs,
	client UTXOClient,
	codec codec.Manager,
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
		utxosBytes, endAddr, endUTXO, err := client.GetAtomicUTXOs(
			ctx,
			addrs,
			sourceChainIDStr,
			fetchLimit,
			startAddr,
			startUTXO,
		)
		if err != nil {
			return err
		}

		for _, utxoBytes := range utxosBytes {
			var utxo lux.UTXO
			_, err := codec.Unmarshal(utxoBytes, &utxo)
			if err != nil {
				return err
			}

			if err := utxos.AddUTXO(ctx, sourceChainID, destinationChainID, &utxo); err != nil {
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

// hexDecode decodes a hex string (with or without 0x prefix)
func hexDecode(s string) ([]byte, error) {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	return hexDecodeString(s)
}

// hexDecodeString decodes a hex string without 0x prefix
func hexDecodeString(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		s = "0" + s
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(b); i++ {
		h := hexValue(s[i*2])
		l := hexValue(s[i*2+1])
		if h == 255 || l == 255 {
			return nil, fmt.Errorf("invalid hex character at position %d", i*2)
		}
		b[i] = h<<4 | l
	}
	return b, nil
}

func hexValue(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	default:
		return 255
	}
}
