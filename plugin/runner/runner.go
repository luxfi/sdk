// (c) 2021-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package runner

import (
	"context"
	"fmt"
	"os"

	"github.com/luxdefi/node/utils/logging"
	"github.com/luxdefi/node/utils/ulimit"
	"github.com/luxdefi/node/vms/rpcchainvm"

	"github.com/luxdefi/evm/plugin/evm"
)

func Run(versionStr string) {
	printVersion, err := PrintVersion()
	if err != nil {
		fmt.Printf("couldn't get config: %s", err)
		os.Exit(1)
	}
	if printVersion && versionStr != "" {
		fmt.Printf(versionStr)
		os.Exit(0)
	}
	if err := ulimit.Set(ulimit.DefaultFDLimit, logging.NoLog{}); err != nil {
		fmt.Printf("failed to set fd limit correctly due to: %s", err)
		os.Exit(1)
	}
	rpcchainvm.Serve(context.Background(), &evm.VM{})
}
