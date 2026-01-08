// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package server

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/luxfi/math/set"
)

const wildcard = "*"

var _ http.Handler = (*allowedHostsHandler)(nil)

func filterInvalidHosts(
	handler http.Handler,
	allowed []string,
) http.Handler {
	s := make(set.Set[string])

	for _, host := range allowed {
		if host == wildcard {
			// wildcards match all hostnames, so just return the base handler
			return handler
		}
		s.Add(strings.ToLower(host))
	}

	return &allowedHostsHandler{
		handler: handler,
		hosts:   s,
	}
}

// allowedHostsHandler is an implementation of http.Handler that validates the
// http host header of incoming requests. This can prevent DNS rebinding attacks
// which do not utilize CORS-headers. Http request host headers are validated
// against a whitelist to determine whether the request should be dropped or
// not.
type allowedHostsHandler struct {
	handler http.Handler
	hosts   set.Set[string]
}

func (a *allowedHostsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// if the host header is missing we can serve this request because dns
	// rebinding attacks rely on this header
	if r.Host == "" {
		a.handler.ServeHTTP(w, r)
		return
	}

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		// either invalid (too many colons) or no port specified
		host = r.Host
	}

	if ipAddr := net.ParseIP(host); ipAddr != nil {
		// accept requests from ips
		a.handler.ServeHTTP(w, r)
		return
	}

	// a specific hostname - we need to check the whitelist to see if we should
	// accept this r
	if a.hosts.Contains(strings.ToLower(host)) {
		a.handler.ServeHTTP(w, r)
		return
	}

	// Return error as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    -32001,
			"message": "invalid host specified",
			"data":    map[string]string{"host": host},
		},
		"id": nil,
	})
}
