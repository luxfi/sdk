// (c) 2021-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"

	"github.com/luxdefi/node/version"
	"github.com/luxdefi/evm/plugin/evm"
	"github.com/luxdefi/evm/plugin/runner"
)

func main() {
<<<<<<< HEAD
	versionString := fmt.Sprintf("Subnet-EVM/%s [Lux Node=%s, rpcchainvm=%d]", evm.Version, version.Current, version.RPCChainVMProtocol)
=======
	versionString := fmt.Sprintf("Subnet-EVM/%s [Luxd=%s, rpcchainvm=%d]", evm.Version, version.Current, version.RPCChainVMProtocol)
>>>>>>> b36c20f (Update executable to luxd)
	runner.Run(versionString)
}
