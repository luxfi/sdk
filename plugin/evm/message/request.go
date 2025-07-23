// (c) 2019-2022, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package message

import (
	"context"
	"fmt"

	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
)

// Request represents a Network request type
type Request interface {
	// Requests should implement String() for logging.
	fmt.Stringer

	// Handle allows `Request` to call respective methods on handler to handle
	// this particular request type
	Handle(ctx context.Context, nodeID interfaces.NodeID, requestID uint32, handler RequestHandler) ([]byte, error)
}

// RequestToBytes marshals the given request object into bytes
func RequestToBytes(codec interfaces.Codec, request Request) ([]byte, error) {
	return interfaces.Marshal(Version, &request)
}
