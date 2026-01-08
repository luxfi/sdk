// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	rpc "github.com/gorilla/rpc/v2/json2"
)

// CleanlyCloseBody drains and closes an HTTP response body to prevent
// HTTP/2 GOAWAY errors caused by closing bodies with unread data.
// See: https://github.com/golang/go/issues/46071
func CleanlyCloseBody(body io.ReadCloser) error {
	if body == nil {
		return nil
	}
	// Drain any remaining data to allow connection reuse
	_, _ = io.Copy(io.Discard, body)
	return body.Close()
}

func SendJSONRequest(
	ctx context.Context,
	uri *url.URL,
	method string,
	params interface{},
	reply interface{},
	options ...Option,
) error {
	requestBodyBytes, err := rpc.EncodeClientRequest(method, params)
	if err != nil {
		return fmt.Errorf("failed to encode client params: %w", err)
	}

	ops := NewOptions(options)
	uri.RawQuery = ops.queryParams.Encode()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		uri.String(),
		bytes.NewBuffer(requestBodyBytes),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	request.Header = ops.headers
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("failed to issue request: %w", err)
	}
	defer CleanlyCloseBody(resp.Body)

	// Return an error for any non successful status code
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("received status code: %d", resp.StatusCode)
	}

	if err := rpc.DecodeClientResponse(resp.Body, reply); err != nil {
		return fmt.Errorf("failed to decode client response: %w", err)
	}
	return nil
}
