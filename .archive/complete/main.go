// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/luxfi/sdk"
	"github.com/luxfi/sdk/blockchain"
	"github.com/luxfi/sdk/chain"
	"github.com/luxfi/sdk/network"
	"github.com/luxfi/sdk/wallet"
	"github.com/luxfi/ids"
	"github.com/luxfi/geth/common"
)

func main() {
	// Initialize the SDK
	luxSDK, err := sdk.New(
		sdk.WithLogLevel("info"),
		sdk.WithDataDir("~/.lux-sdk-example"),
	)
	if err != nil {
		log.Fatal("Failed to initialize SDK:", err)
	}
	defer luxSDK.Close()

	ctx := context.Background()

	// Example 1: Create and manage a network using netrunner
	fmt.Println("\n=== Example 1: Network Management ===")
	if err := networkExample(ctx, luxSDK); err != nil {
		log.Printf("Network example failed: %v", err)
	}

	// Example 2: Build and deploy a blockchain
	fmt.Println("\n=== Example 2: Blockchain Building ===")
	if err := blockchainExample(ctx, luxSDK); err != nil {
		log.Printf("Blockchain example failed: %v", err)
	}

	// Example 3: Staking and validation
	fmt.Println("\n=== Example 3: Staking Operations ===")
	if err := stakingExample(ctx, luxSDK); err != nil {
		log.Printf("Staking example failed: %v", err)
	}

	// Example 4: Asset management on X-Chain
	fmt.Println("\n=== Example 4: Asset Management ===")
	if err := assetExample(ctx, luxSDK); err != nil {
		log.Printf("Asset example failed: %v", err)
	}

	// Example 5: Smart contract deployment on C-Chain
	fmt.Println("\n=== Example 5: Smart Contracts ===")
	if err := smartContractExample(ctx, luxSDK); err != nil {
		log.Printf("Smart contract example failed: %v", err)
	}

	// Example 6: Cross-chain transfers
	fmt.Println("\n=== Example 6: Cross-Chain Operations ===")
	if err := crossChainExample(ctx, luxSDK); err != nil {
		log.Printf("Cross-chain example failed: %v", err)
	}
}

// networkExample demonstrates network creation and management
func networkExample(ctx context.Context, sdk *sdk.LuxSDK) error {
	// Create a local test network
	network, err := sdk.CreateNetwork(ctx, &network.NetworkParams{
		Name:             "test-network",
		Type:             network.NetworkTypeLocal,
		NumNodes:         5,
		EnableStaking:    true,
		EnableMonitoring: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	fmt.Printf("Created network: %s with %d nodes\n", network.Name, len(network.Nodes))

	// Add a new node to the network
	node, err := sdk.networkManager.AddNode(ctx, network.ID, &network.NodeParams{
		Name:        "additional-validator",
		Type:        network.NodeTypeValidator,
		StakeAmount: 2000,
	})
	if err != nil {
		return fmt.Errorf("failed to add node: %w", err)
	}

	fmt.Printf("Added node: %s\n", node.ID)

	// Check node status
	status, err := sdk.networkManager.GetNodeStatus(ctx, network.ID, node.ID)
	if err != nil {
		return fmt.Errorf("failed to get node status: %w", err)
	}

	fmt.Printf("Node status: %s\n", status)

	return nil
}

// blockchainExample demonstrates blockchain creation
func blockchainExample(ctx context.Context, sdk *sdk.LuxSDK) error {
	// Create an L1 blockchain
	l1Chain, err := sdk.CreateL1(ctx, "my-l1-chain", &blockchain.L1Params{
		VMType: blockchain.VMTypeEVM,
		ChainConfig: []byte(`{
			"feeConfig": {
				"gasLimit": 15000000,
				"targetBlockRate": 2,
				"minBaseFee": 25000000000,
				"targetGas": 15000000,
				"baseFeeChangeDenominator": 36,
				"minBlockGasCost": 0,
				"maxBlockGasCost": 1000000,
				"blockGasCostStep": 200000
			}
		}`),
	})
	if err != nil {
		return fmt.Errorf("failed to create L1: %w", err)
	}

	fmt.Printf("Created L1 blockchain: %s\n", l1Chain.Name)

	// Create an L2 rollup
	l2Chain, err := sdk.CreateL2(ctx, "my-l2-rollup", &blockchain.L2Params{
		VMType:          blockchain.VMTypeEVM,
		SequencerType:   "centralized",
		DALayer:         "celestia",
		SettlementChain: l1Chain.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create L2: %w", err)
	}

	fmt.Printf("Created L2 rollup: %s\n", l2Chain.Name)

	// Create an L3 app chain
	l3Chain, err := sdk.CreateL3(ctx, "my-game-chain", &blockchain.L3Params{
		VMType:  blockchain.VMTypeWASM,
		L2Chain: l2Chain.ID,
		AppType: "gaming",
		AppConfig: map[string]interface{}{
			"tickRate":     60,
			"maxPlayers":   1000,
			"assetTypes":   []string{"weapons", "armor", "consumables"},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create L3: %w", err)
	}

	fmt.Printf("Created L3 app chain: %s\n", l3Chain.Name)

	return nil
}

// stakingExample demonstrates staking operations
func stakingExample(ctx context.Context, sdk *sdk.LuxSDK) error {
	// Create a wallet
	w, err := wallet.New("test-wallet")
	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	// Create chain manager
	chainManager, err := chain.NewChainManager("http://localhost:9650", w, sdk.logger)
	if err != nil {
		return fmt.Errorf("failed to create chain manager: %w", err)
	}

	// Stake on primary network
	stakeAmount := big.NewInt(2000) // 2000 LUX
	stakeDuration := 14 * 24 * time.Hour // 14 days

	txID, err := chainManager.Stake(ctx, stakeAmount, stakeDuration)
	if err != nil {
		return fmt.Errorf("failed to stake: %w", err)
	}

	fmt.Printf("Staking transaction: %s\n", txID)

	// Delegate to a validator
	nodeID, _ := ids.NodeIDFromString("NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg")
	delegateAmount := big.NewInt(25) // 25 LUX minimum

	delegateTxID, err := chainManager.Delegate(ctx, nodeID, delegateAmount, stakeDuration)
	if err != nil {
		return fmt.Errorf("failed to delegate: %w", err)
	}

	fmt.Printf("Delegation transaction: %s\n", delegateTxID)

	// Get validators
	validators, err := chainManager.GetValidators(ctx)
	if err != nil {
		return fmt.Errorf("failed to get validators: %w", err)
	}

	fmt.Printf("Current validators: %d, Pending validators: %d\n", 
		len(validators.Current), len(validators.Pending))

	return nil
}

// assetExample demonstrates asset management on X-Chain
func assetExample(ctx context.Context, sdk *sdk.LuxSDK) error {
	// Create a wallet
	w, err := wallet.New("test-wallet")
	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	// Create chain manager
	chainManager, err := chain.NewChainManager("http://localhost:9650", w, sdk.logger)
	if err != nil {
		return fmt.Errorf("failed to create chain manager: %w", err)
	}

	// Create a new asset
	totalSupply := new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e9)) // 1M tokens with 9 decimals
	assetID, err := chainManager.CreateAsset(ctx, "MyToken", "MTK", totalSupply)
	if err != nil {
		return fmt.Errorf("failed to create asset: %w", err)
	}

	fmt.Printf("Created asset: %s\n", assetID)

	// Send some tokens
	recipient, _ := ids.ShortFromString("X-lux1q0qvmc2sfp9c7hgdcxn7xyvj0xjdzj2w8tfgw6")
	sendAmount := big.NewInt(1000 * 1e9) // 1000 tokens

	sendTxID, err := chainManager.SendAsset(ctx, assetID, sendAmount, recipient)
	if err != nil {
		return fmt.Errorf("failed to send asset: %w", err)
	}

	fmt.Printf("Sent %s tokens, tx: %s\n", sendAmount, sendTxID)

	// Create an NFT collection
	nftID, err := chainManager.X().CreateNFT(ctx, &chain.CreateNFTParams{
		Name:   "LuxNFT",
		Symbol: "LNFT",
		Groups: []chain.NFTGroup{
			{
				ID:            0,
				Minters:       []ids.ShortID{w.X().Address()},
				MintThreshold: 1,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create NFT: %w", err)
	}

	fmt.Printf("Created NFT collection: %s\n", nftID)

	return nil
}

// smartContractExample demonstrates C-Chain smart contract operations
func smartContractExample(ctx context.Context, sdk *sdk.LuxSDK) error {
	// Create a wallet
	w, err := wallet.New("test-wallet")
	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	// Create chain manager
	chainManager, err := chain.NewChainManager("http://localhost:9650", w, sdk.logger)
	if err != nil {
		return fmt.Errorf("failed to create chain manager: %w", err)
	}

	// Deploy a simple contract
	// This is example bytecode for a simple storage contract
	bytecode := common.FromHex("0x608060405234801561001057600080fd5b50610150806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80632e64cec11461003b5780636057361d14610059575b600080fd5b610043610075565b60405161005091906100a1565b60405180910390f35b610073600480360381019061006e91906100ed565b61007e565b005b60008054905090565b8060008190555050565b6000819050919050565b61009b81610088565b82525050565b60006020820190506100b66000830184610092565b92915050565b600080fd5b6100ca81610088565b81146100d557600080fd5b50565b6000813590506100e7816100c1565b92915050565b600060208284031215610103576101026100bc565b5b6000610111848285016100d8565b9150509291505056fea2646970667358221220")

	contractAddr, txHash, err := chainManager.C().DeployContract(ctx, &chain.DeployContractParams{
		Bytecode: bytecode,
		GasLimit: 300000,
	})
	if err != nil {
		return fmt.Errorf("failed to deploy contract: %w", err)
	}

	fmt.Printf("Deployed contract at: %s, tx: %s\n", contractAddr.Hex(), txHash.Hex())

	// Wait for deployment
	receipt, err := chainManager.C().WaitForTransaction(ctx, txHash)
	if err != nil {
		return fmt.Errorf("failed to wait for transaction: %w", err)
	}

	if receipt.Status != 1 {
		return fmt.Errorf("contract deployment failed")
	}

	// Call contract method (store a value)
	// Method: store(uint256)
	storeData := common.FromHex("0x6057361d") // store method ID
	value := common.LeftPadBytes(big.NewInt(42).Bytes(), 32)
	storeData = append(storeData, value...)

	storeTx, err := chainManager.C().SendTransaction(ctx, &chain.SendTransactionParams{
		To:       contractAddr,
		Data:     storeData,
		GasLimit: 50000,
	})
	if err != nil {
		return fmt.Errorf("failed to call store: %w", err)
	}

	fmt.Printf("Stored value 42, tx: %s\n", storeTx.Hex())

	// Read contract value (retrieve)
	// Method: retrieve()
	retrieveData := common.FromHex("0x2e64cec1") // retrieve method ID

	result, err := chainManager.C().CallContract(ctx, &chain.CallContractParams{
		To:   contractAddr,
		Data: retrieveData,
	})
	if err != nil {
		return fmt.Errorf("failed to call retrieve: %w", err)
	}

	storedValue := new(big.Int).SetBytes(result)
	fmt.Printf("Retrieved value: %s\n", storedValue)

	return nil
}

// crossChainExample demonstrates cross-chain transfers
func crossChainExample(ctx context.Context, sdk *sdk.LuxSDK) error {
	// Create a wallet
	w, err := wallet.New("test-wallet")
	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	// Create chain manager
	chainManager, err := chain.NewChainManager("http://localhost:9650", w, sdk.logger)
	if err != nil {
		return fmt.Errorf("failed to create chain manager: %w", err)
	}

	// Transfer LUX from C-Chain to P-Chain
	transferAmount := big.NewInt(100 * 1e9) // 100 LUX
	
	transferTxID, err := chainManager.TransferCrossChain(ctx, &chain.CrossChainTransferParams{
		SourceChain: "C",
		TargetChain: "P",
		AssetID:     ids.Empty, // LUX
		Amount:      transferAmount,
		To:          w.P().Address(),
	})
	if err != nil {
		return fmt.Errorf("failed to transfer cross-chain: %w", err)
	}

	fmt.Printf("Cross-chain transfer completed: %s\n", transferTxID)

	// Get balances across all chains
	balances, err := chainManager.GetBalance(ctx, w.P().Address().String())
	if err != nil {
		return fmt.Errorf("failed to get balances: %w", err)
	}

	fmt.Println("Balances across chains:")
	for chain, balance := range balances.Chains {
		if balance.LUX != nil {
			fmt.Printf("  %s-Chain: %s LUX\n", chain, balance.LUX)
		}
		for assetID, amount := range balance.Assets {
			fmt.Printf("  %s-Chain: %s of asset %s\n", chain, amount, assetID)
		}
	}

	return nil
}