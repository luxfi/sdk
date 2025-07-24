// (c) 2019-2020, Lux Industries, Inc.
// All rights reserved.
// See the file LICENSE for licensing terms.

package consensus

import (
	"github.com/luxfi/node/api/metrics"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/utils/crypto/bls"
	"github.com/luxfi/node/utils/logging"
)

// Context is the context for the chain
type Context struct {
	NetworkID    uint32
	SubnetID     ids.ID
	ChainID      ids.ID
	NodeID       ids.NodeID
	PublicKey    *bls.PublicKey
	XChainID     ids.ID
	CChainID     ids.ID
	LUXAssetID   ids.ID
	Log          logging.Logger
	Metrics      metrics.MultiGatherer
	ChainDataDir string
	AliasManager AliasManager
	Validators   ValidatorManager
}

// AliasManager interface
type AliasManager interface {
	Alias(ids.ID, string) error
	Aliases(ids.ID) ([]string, error)
	PrimaryAlias(ids.ID) (string, error)
}

// ValidatorManager interface
type ValidatorManager interface {
	GetValidator(ids.NodeID) (*Validator, bool)
	GetValidatorSet() map[ids.NodeID]*Validator
}

// Validator represents a validator
type Validator struct {
	NodeID ids.NodeID
	Weight uint64
}
