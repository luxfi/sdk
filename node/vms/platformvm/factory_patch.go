// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//go:build singlevalidator

package platformvm

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/version"
	"github.com/luxfi/sdk/node/vms"
)

// SingleValidatorFactory creates a mock Platform VM for single validator mode
type SingleValidatorFactory struct{}

func (f *SingleValidatorFactory) New(vms.Config) (interface{}, error) {
	// Return a minimal implementation that doesn't require multiple validators
	return &singleValidatorVM{}, nil
}

type singleValidatorVM struct{}

func (vm *singleValidatorVM) Initialize(
	ctx interface{},
	dbManager interface{},
	genesisBytes []byte,
	upgradeBytes []byte,
	configBytes []byte,
	msgChan chan interface{},
	fxs []interface{},
	appSender interface{},
) error {
	// Initialize with single validator mode
	return nil
}

func (vm *singleValidatorVM) Bootstrapping() error     { return nil }
func (vm *singleValidatorVM) Bootstrapped() error      { return nil }
func (vm *singleValidatorVM) Shutdown() error          { return nil }
func (vm *singleValidatorVM) Version() (string, error) { return "single-validator-1.0", nil }
func (vm *singleValidatorVM) CreateHandlers() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (vm *singleValidatorVM) CreateStaticHandlers() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (vm *singleValidatorVM) Connected(ids.NodeID, *version.Application) error { return nil }
func (vm *singleValidatorVM) Disconnected(ids.NodeID) error                    { return nil }
