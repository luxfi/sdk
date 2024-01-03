// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"github.com/luxdefi/node/api/metrics"
	"github.com/luxdefi/node/ids"
	"github.com/luxdefi/node/snow"
	"github.com/luxdefi/node/utils/crypto/bls"
	"github.com/luxdefi/node/utils/logging"
)

func TestSnowContext() *snow.Context {
	sk, err := bls.NewSecretKey()
	if err != nil {
		panic(err)
	}
	pk := bls.PublicFromSecretKey(sk)
	return &snow.Context{
		NetworkID:    0,
		SubnetID:     ids.Empty,
		ChainID:      ids.Empty,
		NodeID:       ids.EmptyNodeID,
		PublicKey:    pk,
		Log:          logging.NoLog{},
		BCLookup:     ids.NewAliaser(),
		Metrics:      metrics.NewOptionalGatherer(),
		ChainDataDir: "",
	}
}
