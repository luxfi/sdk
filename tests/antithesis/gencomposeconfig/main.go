// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/luxfi/luxd/tests/antithesis"
	"github.com/luxfi/luxd/tests/fixture/tmpnet"

	"github.com/luxfi/evm/tests/utils"
)

const baseImageName = "antithesis-evm"

// Creates docker-compose.yml and its associated volumes in the target path.
func main() {
	// Assume the working directory is the root of the repository
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current working directory: %s", err)
	}

	genesisPath := filepath.Join(cwd, "tests/load/genesis/genesis.json")

	// Create a network with a evm subnet
	network := tmpnet.LocalNetworkOrPanic()
	network.Subnets = []*tmpnet.Subnet{
		utils.NewTmpnetSubnet("evm", genesisPath, utils.DefaultChainConfig, network.Nodes...),
	}

	if err := antithesis.GenerateComposeConfig(network, baseImageName); err != nil {
		log.Fatalf("failed to generate compose config: %v", err)
	}
}
