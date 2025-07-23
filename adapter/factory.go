// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package adapter

import (
	"github.com/luxfi/evm/interfaces"
	
	// Node imports for actual implementations
	nodedb "github.com/luxfi/evm/interfaces"
	nodeconsensus "github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
)

// Factory provides methods to create adapted interfaces from node types
type Factory struct{}

// NewFactory creates a new adapter factory
func NewFactory() *Factory {
	return &Factory{}
}

// AdaptDatabase adapts a node database to the interface
func (f *Factory) AdaptDatabase(db interfaces.Database) interfaces.Database {
	return NewDatabaseAdapter(db)
}

// AdaptBlock adapts a node block to the interface
func (f *Factory) AdaptBlock(block interfaces.Block) interfaces.NodeBlock {
	return NewBlockAdapter(block)
}

// AdaptConsensus adapts node consensus to the interface
func (f *Factory) AdaptConsensus(consensus *nodeinterfaces.ChainContext) interfaces.NodeConsensus {
	return NewConsensusAdapter(consensus)
}

// AdaptValidatorState adapts node validator state to the interface
func (f *Factory) AdaptValidatorState(state interfaces.State) interfaces.ValidatorState {
	return NewValidatorStateAdapter(state)
}

// AdaptWarpBackend adapts node warp backend to the interface
func (f *Factory) AdaptWarpBackend(backend interfaces.Backend) interfaces.WarpBackend {
	return NewWarpBackendAdapter(backend)
}