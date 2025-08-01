// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"

	"github.com/luxfi/node/version"
	"github.com/luxfi/evm/plugin/evm"
	"github.com/luxfi/evm/plugin/runner"
)

func main() {
	versionString := fmt.Sprintf("Subnet-EVM/%s [Luxd=%s, rpcchainvm=%d]", evm.Version, version.Current, version.RPCChainVMProtocol)
	runner.Run(versionString)
}
