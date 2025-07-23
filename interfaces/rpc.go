// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package interfaces

import (
	"context"
	"net/http"
)

// RPCOption configures an RPC request
type RPCOption func(*RPCOptions)

// RPCOptions holds RPC request options
type RPCOptions struct {
	Headers http.Header
}

// EndpointRequester makes requests to RPC endpoints
type EndpointRequester interface {
	// SendRequest sends an RPC request
	SendRequest(ctx context.Context, method string, params interface{}, reply interface{}, options ...RPCOption) error
}

// NewEndpointRequester creates a new endpoint requester
func NewEndpointRequester(uri, endpoint string) EndpointRequester {
	return &endpointRequester{
		uri:      uri,
		endpoint: endpoint,
	}
}

// endpointRequester implements EndpointRequester
type endpointRequester struct {
	uri      string
	endpoint string
}

// SendRequest implements EndpointRequester
func (e *endpointRequester) SendRequest(ctx context.Context, method string, params interface{}, reply interface{}, options ...RPCOption) error {
	// For now, return nil as we're just creating the interface
	// In a real implementation, this would make the actual RPC call
	return nil
}