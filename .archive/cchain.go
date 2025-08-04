// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/wallet"
	"github.com/luxfi/log"
)

// CChainClient handles all C-Chain (EVM) operations
type CChainClient struct {
	client   *ethclient.Client
	wallet   *wallet.Wallet
	logger   log.Logger
	endpoint string
	chainID  *big.Int
}

// NewCChainClient creates a new C-Chain client
func NewCChainClient(endpoint string, wallet *wallet.Wallet, logger log.Logger) (*CChainClient, error) {
	client, err := ethclient.Dial(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to C-Chain: %w", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	return &CChainClient{
		client:   client,
		wallet:   wallet,
		logger:   logger,
		endpoint: endpoint,
		chainID:  chainID,
	}, nil
}

// Transaction Operations

// SendTransaction sends a transaction on C-Chain
func (c *CChainClient) SendTransaction(ctx context.Context, params *SendTransactionParams) (common.Hash, error) {
	c.logger.Info("sending transaction",
		"to", params.To,
		"value", params.Value,
		"gasLimit", params.GasLimit,
	)

	nonce, err := c.client.PendingNonceAt(ctx, params.From)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice := params.GasPrice
	if gasPrice == nil {
		gasPrice, err = c.client.SuggestGasPrice(ctx)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to suggest gas price: %w", err)
		}
	}

	tx := types.NewTransaction(
		nonce,
		params.To,
		params.Value,
		params.GasLimit,
		gasPrice,
		params.Data,
	)

	signedTx, err := c.wallet.C().SignTx(tx, c.chainID)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	if err := c.client.SendTransaction(ctx, signedTx); err != nil {
		return common.Hash{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx.Hash(), nil
}

// DeployContract deploys a smart contract
func (c *CChainClient) DeployContract(ctx context.Context, params *DeployContractParams) (common.Address, common.Hash, error) {
	c.logger.Info("deploying contract", "bytecodeSize", len(params.Bytecode))

	from := params.From
	if from == (common.Address{}) {
		from = c.wallet.C().Address()
	}

	nonce, err := c.client.PendingNonceAt(ctx, from)
	if err != nil {
		return common.Address{}, common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Address{}, common.Hash{}, fmt.Errorf("failed to suggest gas price: %w", err)
	}

	// Append constructor arguments if provided
	data := params.Bytecode
	if len(params.ConstructorArgs) > 0 {
		data = append(data, params.ConstructorArgs...)
	}

	tx := types.NewContractCreation(
		nonce,
		params.Value,
		params.GasLimit,
		gasPrice,
		data,
	)

	signedTx, err := c.wallet.C().SignTx(tx, c.chainID)
	if err != nil {
		return common.Address{}, common.Hash{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	if err := c.client.SendTransaction(ctx, signedTx); err != nil {
		return common.Address{}, common.Hash{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Calculate contract address
	contractAddr := crypto.CreateAddress(from, nonce)

	return contractAddr, signedTx.Hash(), nil
}

// CallContract calls a contract method
func (c *CChainClient) CallContract(ctx context.Context, params *CallContractParams) ([]byte, error) {
	msg := ethereum.CallMsg{
		From:     params.From,
		To:       &params.To,
		Gas:      params.GasLimit,
		GasPrice: params.GasPrice,
		Value:    params.Value,
		Data:     params.Data,
	}

	result, err := c.client.CallContract(ctx, msg, params.BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	return result, nil
}

// DeFi Operations

// SwapTokens performs a token swap using a DEX
func (c *CChainClient) SwapTokens(ctx context.Context, params *SwapParams) (common.Hash, error) {
	c.logger.Info("swapping tokens",
		"tokenIn", params.TokenIn,
		"tokenOut", params.TokenOut,
		"amountIn", params.AmountIn,
	)

	// This would interact with a DEX contract like Uniswap
	// For now, return a placeholder
	return common.Hash{}, fmt.Errorf("swap functionality not yet implemented")
}

// ProvideLiquidity adds liquidity to a pool
func (c *CChainClient) ProvideLiquidity(ctx context.Context, params *LiquidityParams) (common.Hash, error) {
	c.logger.Info("providing liquidity",
		"tokenA", params.TokenA,
		"tokenB", params.TokenB,
		"amountA", params.AmountA,
		"amountB", params.AmountB,
	)

	// This would interact with a DEX contract
	return common.Hash{}, fmt.Errorf("liquidity functionality not yet implemented")
}

// Stake stakes tokens in a DeFi protocol
func (c *CChainClient) StakeTokens(ctx context.Context, params *StakeTokensParams) (common.Hash, error) {
	c.logger.Info("staking tokens",
		"token", params.Token,
		"amount", params.Amount,
		"pool", params.Pool,
	)

	// This would interact with a staking contract
	return common.Hash{}, fmt.Errorf("staking functionality not yet implemented")
}

// Cross-Chain Operations

// Export exports LUX from C-Chain
func (c *CChainClient) Export(ctx context.Context, amount *big.Int, to ids.ShortID, targetChain ids.ID) (common.Hash, error) {
	c.logger.Info("exporting from C-Chain",
		"amount", amount,
		"to", to,
		"targetChain", targetChain,
	)

	// This would interact with the export precompile
	return common.Hash{}, fmt.Errorf("export functionality not yet implemented")
}

// Import imports LUX to C-Chain
func (c *CChainClient) Import(ctx context.Context, sourceChain ids.ID, to ids.ShortID) (common.Hash, error) {
	c.logger.Info("importing to C-Chain",
		"sourceChain", sourceChain,
		"to", to,
	)

	// This would interact with the import precompile
	return common.Hash{}, fmt.Errorf("import functionality not yet implemented")
}

// Query Operations

// GetBalance returns the balance of an address
func (c *CChainClient) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	addr := common.HexToAddress(address)
	balance, err := c.client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// GetTokenBalance returns the token balance of an address
func (c *CChainClient) GetTokenBalance(ctx context.Context, token, holder common.Address) (*big.Int, error) {
	// ERC20 balanceOf method ID
	balanceOfMethod := "0x70a08231"
	
	data := common.FromHex(balanceOfMethod)
	data = append(data, common.LeftPadBytes(holder.Bytes(), 32)...)

	msg := ethereum.CallMsg{
		To:   &token,
		Data: data,
	}

	result, err := c.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get token balance: %w", err)
	}

	balance := new(big.Int).SetBytes(result)
	return balance, nil
}

// GetTransactionReceipt returns a transaction receipt
func (c *CChainClient) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	receipt, err := c.client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	return receipt, nil
}

// GetBlockNumber returns the current block number
func (c *CChainClient) GetBlockNumber(ctx context.Context) (uint64, error) {
	blockNumber, err := c.client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get block number: %w", err)
	}

	return blockNumber, nil
}

// GetGasPrice returns the current gas price
func (c *CChainClient) GetGasPrice(ctx context.Context) (*big.Int, error) {
	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	return gasPrice, nil
}

// EstimateGas estimates gas for a transaction
func (c *CChainClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	gas, err := c.client.EstimateGas(ctx, msg)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate gas: %w", err)
	}

	return gas, nil
}

// WaitForTransaction waits for a transaction to be mined
func (c *CChainClient) WaitForTransaction(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	c.logger.Info("waiting for transaction", "hash", txHash.Hex())

	for {
		receipt, err := c.client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
			// Continue polling
		}
	}
}

// Parameter types

type SendTransactionParams struct {
	From     common.Address
	To       common.Address
	Value    *big.Int
	Data     []byte
	GasLimit uint64
	GasPrice *big.Int
}

type DeployContractParams struct {
	From            common.Address
	Bytecode        []byte
	ConstructorArgs []byte
	Value           *big.Int
	GasLimit        uint64
}

type CallContractParams struct {
	From        common.Address
	To          common.Address
	Data        []byte
	Value       *big.Int
	GasLimit    uint64
	GasPrice    *big.Int
	BlockNumber *big.Int
}

type SwapParams struct {
	TokenIn      common.Address
	TokenOut     common.Address
	AmountIn     *big.Int
	AmountOutMin *big.Int
	Path         []common.Address
	To           common.Address
	Deadline     *big.Int
}

type LiquidityParams struct {
	TokenA        common.Address
	TokenB        common.Address
	AmountA       *big.Int
	AmountB       *big.Int
	AmountAMin    *big.Int
	AmountBMin    *big.Int
	To            common.Address
	Deadline      *big.Int
}

type StakeTokensParams struct {
	Token  common.Address
	Amount *big.Int
	Pool   common.Address
}