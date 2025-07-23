// (c) 2024, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"os"
	"testing"

	"github.com/luxfi/evm/log"
	"github.com/stretchr/testify/require"
)

func TestInitLogger(t *testing.T) {
	require := require.New(t)
	_, err := InitLogger("alias", "info", true, os.Stderr)
	require.NoError(err)
	log.Info("test")
}
