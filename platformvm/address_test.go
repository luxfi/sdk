// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/constants"
)

// TestTheNetSelectorIsSpelledNetID is the defect that did not fail.
//
// vms/platformvm/service.go reads netID. Eleven types here sent and read
// chainID, and nothing errored: the node found no netID, left the id empty, and
// an empty id is the primary network. So asking an L1 for its validators, its
// stake, its supply or its minimum stake quietly answered for the primary
// network instead, and the caller had no way to tell.
//
// It is asserted on the encoded bytes, because what the node reads is the wire
// and not the struct.
func TestTheNetSelectorIsSpelledNetID(t *testing.T) {
	require := require.New(t)
	net := ids.GenerateTestID()

	for name, args := range map[string]any{
		"getNet":               &GetNetArgs{ChainID: net},
		"getStakingAssetID":    &GetStakingAssetIDArgs{ChainID: net},
		"getCurrentValidators": &GetCurrentValidatorsArgs{ChainID: net},
		"getCurrentSupply":     &GetCurrentSupplyArgs{ChainID: net},
		"sampleValidators":     &SampleValidatorsArgs{ChainID: net},
		"validates":            &ValidatesArgs{ChainID: net},
		"getMinStake":          &GetMinStakeArgs{ChainID: net},
		"getTotalStake":        &GetTotalStakeArgs{ChainID: net},
		"getValidatorsAt":      &GetValidatorsAtArgs{ChainID: net},
	} {
		b, err := json.Marshal(args)
		require.NoError(err, name)
		require.Contains(string(b), `"netID"`, "%s does not send netID: %s", name, b)
		require.NotContains(string(b), `"chainID"`, "%s still sends chainID: %s", name, b)
	}

	// A reply is read the same way. GetL1ValidatorReply is the one that was
	// wrong; ValidatedByResponse was already right and is here so a later edit
	// cannot quietly take it back.
	var l1 GetL1ValidatorReply
	require.NoError(json.Unmarshal([]byte(`{"netID":"`+net.String()+`"}`), &l1))
	require.Equal(net, l1.ChainID, "GetL1ValidatorReply does not read netID")

	var by ValidatedByResponse
	require.NoError(json.Unmarshal([]byte(`{"netID":"`+net.String()+`"}`), &by))
	require.Equal(net, by.ChainID, "ValidatedByResponse does not read netID")
}

// TestTheAddressIsOneTheNodeServes pins the prefix. The node serves /v1 and
// nothing else — "the legacy /ext prefix is gone", server/http/server.go — and
// this package asked for /ext/P, so every P-chain call was a 404.
func TestTheAddressIsOneTheNodeServes(t *testing.T) {
	require := require.New(t)

	addr := constants.Chain("http://localhost:9650", "P")
	require.True(strings.HasPrefix(addr, "http://localhost:9650/v1/"), addr)
	require.NotContains(addr, "/ext/")
	require.True(strings.HasSuffix(addr, "/P"), addr)
}
