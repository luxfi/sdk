// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"

	"github.com/luxdefi/node/version"
	"github.com/luxdefi/evm/plugin/evm"
	"github.com/luxdefi/evm/plugin/runner"
)

func main() {
	versionString := fmt.Sprintf("EVM/%s [Lux=%s, rpcchainvm=%d]", evm.Version, version.Current, version.RPCChainVMProtocol)
	runner.Run(versionString)
}
