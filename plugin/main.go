// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/luxfi/evm/plugin/evm"
	"github.com/luxfi/evm/plugin/runner"
	"github.com/luxfi/node/version"
)

func main() {
	// DEBUG: Write to file to verify plugin is starting
	if f, err := os.OpenFile("/tmp/evm_plugin_start.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(f, "[%s] EVM plugin main() started, PID=%d\n", time.Now().Format(time.RFC3339), os.Getpid())
		f.Close()
	}
	versionString := fmt.Sprintf("Subnet-EVM/%s [Luxd=%s, rpcchainvm=%d]", evm.Version, version.Current, version.RPCChainVMProtocol)
	runner.Run(versionString)
}
