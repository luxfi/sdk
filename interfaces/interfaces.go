// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"github.com/luxfi/node/ids"
)

// Manager handles validator management
type Manager interface {
	Connected(nodeID ids.NodeID)
	Disconnect(nodeID ids.NodeID) error
}
