// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package runner

import (
	"flag"
	"os"
)

const versionKey = "version"

func PrintVersion() (bool, error) {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	versionFlag := fs.Bool(versionKey, false, "print version")
	
	if err := fs.Parse(os.Args[1:]); err != nil {
		return false, err
	}

	return *versionFlag, nil
}