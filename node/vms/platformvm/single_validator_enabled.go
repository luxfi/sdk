// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

// SingleValidatorMode enables single validator operation without build tags
var SingleValidatorMode = false

func init() {
	if SingleValidatorMode {
		// Override any validator count requirements
		MinValidatorCount = 1
		RequireValidatorApproval = false
	}
}

// MinValidatorCount sets the minimum number of validators required
var MinValidatorCount = 3

// RequireValidatorApproval determines if multiple validators must approve
var RequireValidatorApproval = true
