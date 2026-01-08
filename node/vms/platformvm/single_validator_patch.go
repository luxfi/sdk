// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build singlevalidator

package platformvm

import "github.com/luxfi/sdk/node/vms/platformvm/config"

// EnableSingleValidatorMode allows Platform VM to run with a single validator
func init() {
	// Override minimum validator requirements
	config.MinValidatorCount = 1
}

// SingleValidatorFactory creates Platform VM that accepts single validator
type SingleValidatorFactory struct {
	Factory
}

// New returns Platform VM configured for single validator operation
func (f *SingleValidatorFactory) New(logger interface{}) (interface{}, error) {
	vm, err := f.Factory.New(logger)
	if err != nil {
		return nil, err
	}

	// Configure VM for single validator
	if pvm, ok := vm.(*VM); ok {
		// Override validation checks that require multiple validators
		pvm.Config.MinValidators = 1
		pvm.Config.RequireValidatorApproval = false
	}

	return vm, nil
}
