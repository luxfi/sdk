// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package health

import sdkhealth "github.com/luxfi/sdk/api/health"

// notYetRunResult is the result that is returned when a HealthCheck hasn't been
// run yet.
var notYetRunResult Result

func init() {
	err := "not yet run"
	notYetRunResult = Result{
		Error: &err,
	}
}

type Result = sdkhealth.Result
