// Package nodevalidators contains minimal validator types vendored from the
// node/consensus/validators package to decouple EVM validator tests.
package nodevalidators

import "github.com/luxfi/node/ids"

// GetCurrentValidatorOutput is the minimal subset of validator metadata
// returned by GetCurrentValidatorSet that EVM needs to load validators.
type GetCurrentValidatorOutput struct {
   NodeID        ids.NodeID
   Weight        uint64
   StartTime     uint64
   IsActive      bool
   IsL1Validator bool
}
