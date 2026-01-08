// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/const"
)

func TestChainValidatorVerifyChainID(t *testing.T) {
	require := require.New(t)

	// Error path
	{
		vdr := &ChainValidator{
			Chain: constants.PrimaryNetworkID,
		}

		err := vdr.Verify()
		require.ErrorIs(err, errBadChainID)
	}

	// Happy path
	{
		vdr := &ChainValidator{
			Chain: ids.GenerateTestID(),
			Validator: Validator{
				Wght: 1,
			},
		}

		require.NoError(vdr.Verify())
	}
}
