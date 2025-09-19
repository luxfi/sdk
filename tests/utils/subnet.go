// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	// "time" // Commented out - used in commented code

	// "github.com/luxfi/ids" // Commented out - used in commented code
	"github.com/luxfi/node/api/health"
	// "github.com/luxfi/node/api/info" // Commented out - used in commented code
	// "github.com/luxfi/node/genesis" // Commented out - used in commented code
	// "github.com/luxfi/node/vms/secp256k1fx" // Commented out - used in commented code
	// "github.com/luxfi/node/wallet/net/primary" // TODO: This package doesn't exist in v1.16.15
	"github.com/go-cmd/cmd"
	// "github.com/luxfi/evm/core" // Commented out - used in commented code
	// "github.com/luxfi/evm/plugin/evm" // Commented out - used in commented code
	"github.com/luxfi/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/require"
)

type SubnetSuite struct {
	blockchainIDs map[string]string
	lock          sync.RWMutex
}

func (s *SubnetSuite) GetBlockchainID(alias string) string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.blockchainIDs[alias]
}

func (s *SubnetSuite) SetBlockchainIDs(blockchainIDs map[string]string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.blockchainIDs = blockchainIDs
}

// CreateSubnetsSuite creates subnets for given [genesisFiles], and registers a before suite that starts an Luxd process to use for the e2e tests.
// genesisFiles is a map of test aliases to genesis file paths.
func CreateSubnetsSuite(genesisFiles map[string]string) *SubnetSuite {
	require := require.New(ginkgo.GinkgoT())

	// Keep track of the Luxd external bash script, it is null for most
	// processes except the first process that starts Luxd
	var startCmd *cmd.Cmd

	// This is used to pass the blockchain IDs from the SynchronizedBeforeSuite() to the tests
	var globalSuite SubnetSuite

	// Our test suite runs in separate processes, ginkgo has
	// SynchronizedBeforeSuite() which runs once, and its return value is passed
	// over to each worker.
	//
	// Here an Luxd node instance is started, and subnets are created for
	// each test case. Each test case has its own subnet, therefore all tests
	// can run in parallel without any issue.
	//
	_ = ginkgo.SynchronizedBeforeSuite(func() []byte {
		ctx, cancel := context.WithTimeout(context.Background(), BootLuxNodeTimeout)
		defer cancel()

		wd, err := os.Getwd()
		require.NoError(err)
		log.Info("Starting Luxd node", "wd", wd)
		cmd, err := RunCommand("./scripts/run.sh")
		require.NoError(err)
		startCmd = cmd

		// Assumes that startCmd will launch a node with HTTP Port at [utils.DefaultLocalNodeURI]
		healthClient := health.NewClient(DefaultLocalNodeURI)
		healthy, err := health.AwaitReady(ctx, healthClient, HealthCheckTimeout, nil)
		require.NoError(err)
		require.True(healthy)
		log.Info("Luxd node is healthy")

		blockchainIDs := make(map[string]string)
		for alias, file := range genesisFiles {
			blockchainIDs[alias] = CreateNewSubnet(ctx, file)
		}

		blockchainIDsBytes, err := json.Marshal(blockchainIDs)
		require.NoError(err)
		return blockchainIDsBytes
	}, func(ctx ginkgo.SpecContext, data []byte) {
		blockchainIDs := make(map[string]string)
		require.NoError(json.Unmarshal(data, &blockchainIDs))

		globalSuite.SetBlockchainIDs(blockchainIDs)
	})

	// SynchronizedAfterSuite() takes two functions, the first runs after each test suite is done and the second
	// function is executed once when all the tests are done. This function is used
	// to gracefully shutdown the Luxd node.
	_ = ginkgo.SynchronizedAfterSuite(func() {}, func() {
		require.NotNil(startCmd)
		require.NoError(startCmd.Stop())
	})

	return &globalSuite
}

// CreateNewSubnet creates a new subnet and Subnet-EVM blockchain with the given genesis file.
// returns the ID of the new created blockchain.
// TODO: This function is disabled because wallet/net/primary package doesn't exist in node v1.16.15
func CreateNewSubnet(ctx context.Context, genesisFilePath string) string {
	panic("CreateNewSubnet is currently disabled - wallet/net/primary package not available in node v1.16.15")
	// require := require.New(ginkgo.GinkgoT())

	// kc := secp256k1fx.NewKeychain(genesis.EWOQKey)

	// // MakeWallet fetches the available UTXOs owned by [kc] on the network
	// // that [LocalAPIURI] is hosting.
	// walletConfig := &primary.WalletConfig{
	// 	URI: DefaultLocalNodeURI,
	// 	LUXKeychain: kc,
	// 	EthKeychain: kc,
	// }
	// wallet, err := primary.MakeWallet(ctx, walletConfig)
	// require.NoError(err)

	// pWallet := wallet.P()

	// owner := &secp256k1fx.OutputOwners{
	// 	Threshold: 1,
	// 	Addrs: []ids.ShortID{
	// 		genesis.EWOQKey.PublicKey().Address(),
	// 	},
	// }

	// wd, err := os.Getwd()
	// require.NoError(err)
	// log.Info("Reading genesis file", "filePath", genesisFilePath, "wd", wd)
	// genesisBytes, err := os.ReadFile(genesisFilePath)
	// require.NoError(err)

	// log.Info("Creating new subnet")
	// createNetTx, err := pWallet.IssueCreateNetTx(owner)
	// require.NoError(err)

	// genesis := &core.Genesis{}
	// require.NoError(json.Unmarshal(genesisBytes, genesis))

	// log.Info("Creating new Subnet-EVM blockchain", "genesis", genesis)
	// createChainTx, err := pWallet.IssueCreateChainTx(
	// 	createNetTx.ID(),
	// 	genesisBytes,
	// 	evm.ID,
	// 	nil,
	// 	"testChain",
	// )
	// require.NoError(err)
	// createChainTxID := createChainTx.ID()

	// // Confirm the new blockchain is ready by waiting for the readiness endpoint
	// infoClient := info.NewClient(DefaultLocalNodeURI)
	// bootstrapped, err := info.AwaitBootstrapped(ctx, infoClient, createChainTxID.String(), 2*time.Second)
	// require.NoError(err)
	// require.True(bootstrapped)

	// // Return the blockchainID of the newly created blockchain
	// return createChainTxID.String()
}

// GetDefaultChainURI returns the default chain URI for a given blockchainID
func GetDefaultChainURI(blockchainID string) string {
	return fmt.Sprintf("%s/ext/bc/%s/rpc", DefaultLocalNodeURI, blockchainID)
}

// GetFilesAndAliases returns a map of aliases to file paths in given [dir].
func GetFilesAndAliases(dir string) (map[string]string, error) {
	files, err := filepath.Glob(dir)
	if err != nil {
		return nil, err
	}
	aliasesToFiles := make(map[string]string)
	for _, file := range files {
		alias := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		aliasesToFiles[alias] = file
	}
	return aliasesToFiles, nil
}
