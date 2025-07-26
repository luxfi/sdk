// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build minimal
// +build minimal

package runner

import (
	"context"
	"fmt"
	"os"

	"github.com/luxfi/node/utils/logging"
	"github.com/luxfi/node/utils/ulimit"
	"github.com/luxfi/node/vms/rpcchainvm"
	
	"github.com/luxfi/evm/plugin/evm"
)

func Run(versionStr string) {
	printVersion, err := PrintVersion()
	if err != nil {
		fmt.Printf("couldn't get config: %s", err)
		os.Exit(1)
	}
	if printVersion && versionStr != "" {
		fmt.Println(versionStr)
		os.Exit(0)
	}
	
	// Set file descriptor limit
	if err := ulimit.Set(ulimit.DefaultFDLimit, logging.NoLog{}); err != nil {
		fmt.Printf("failed to set fd limit correctly due to: %s", err)
		os.Exit(1)
	}
	
	// Create minimal VM instance and serve it via RPC
	vm := &evm.MinimalVM{}
	rpcchainvm.Serve(context.Background(), vm)
}