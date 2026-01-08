// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.


package server

import "net/http"

type Wrapper interface {
	// WrapHandler wraps an http.Handler.
	WrapHandler(h http.Handler) http.Handler
}
