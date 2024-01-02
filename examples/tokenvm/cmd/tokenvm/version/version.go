// Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/luxdefi/vmsdk/examples/tokenvm/consts"
	"github.com/luxdefi/vmsdk/examples/tokenvm/version"
)

func init() {
	cobra.EnablePrefixMatching = true
}

// NewCommand implements "tokenvm version" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints out the verson",
		RunE:  versionFunc,
	}
	return cmd
}

func versionFunc(*cobra.Command, []string) error {
	fmt.Printf("%s@%s (%s)\n", consts.Name, version.Version, consts.ID)
	return nil
}
