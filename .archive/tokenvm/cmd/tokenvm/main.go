// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/luxfi/log"
	"github.com/luxfi/sdk/examples/tokenvm/cmd/tokenvm/version"
	"github.com/luxfi/sdk/examples/tokenvm/controller"
	"github.com/luxfi/ulimit"
	"github.com/luxfi/vms/rpcchainvm"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:        "tokenvm",
	Short:      "TokenVM agent",
	SuggestFor: []string{"tokenvm"},
	RunE:       runFunc,
}

func init() {
	cobra.EnablePrefixMatching = true
}

func init() {
	rootCmd.AddCommand(
		version.NewCommand(),
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "tokenvm failed %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func runFunc(*cobra.Command, []string) error {
	if err := ulimit.Set(ulimit.DefaultFDLimit, logging.NoLog{}); err != nil {
		return fmt.Errorf("%w: failed to set fd limit correctly", err)
	}
	return rpcchainvm.Serve(context.TODO(), controller.New())
}
