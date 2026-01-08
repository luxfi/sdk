// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

const (
	// First primary network apricot delegators are moved from the pending to
	// the current validator set,
	PrimaryNetworkDelegatorApricotPendingPriority Priority = iota + 1
	// then primary network validators,
	PrimaryNetworkValidatorPendingPriority
	// then primary network banff delegators,
	PrimaryNetworkDelegatorBanffPendingPriority
	// then permissionless chain validators,
	ChainPermissionlessValidatorPendingPriority
	// then permissionless chain delegators.
	ChainPermissionlessDelegatorPendingPriority
	// then permissioned chain validators,
	ChainPermissionedValidatorPendingPriority

	// First permissioned chain validators are removed from the current
	// validator set,
	// Invariant: All permissioned stakers must be removed first because they
	//            are removed by the advancement of time. Permissionless stakers
	//            are removed with a RewardValidatorTx after time has advanced.
	ChainPermissionedValidatorCurrentPriority
	// then permissionless chain delegators,
	ChainPermissionlessDelegatorCurrentPriority
	// then permissionless chain validators,
	ChainPermissionlessValidatorCurrentPriority
	// then primary network delegators,
	PrimaryNetworkDelegatorCurrentPriority
	// then primary network validators.
	PrimaryNetworkValidatorCurrentPriority
)

// Deprecated: Use Chain* priority constants instead
const (
	NetPermissionlessValidatorPendingPriority = ChainPermissionlessValidatorPendingPriority
	NetPermissionlessDelegatorPendingPriority = ChainPermissionlessDelegatorPendingPriority
	NetPermissionedValidatorPendingPriority   = ChainPermissionedValidatorPendingPriority
	NetPermissionedValidatorCurrentPriority   = ChainPermissionedValidatorCurrentPriority
	NetPermissionlessDelegatorCurrentPriority = ChainPermissionlessDelegatorCurrentPriority
	NetPermissionlessValidatorCurrentPriority = ChainPermissionlessValidatorCurrentPriority
)

var PendingToCurrentPriorities = []Priority{
	PrimaryNetworkDelegatorApricotPendingPriority:   PrimaryNetworkDelegatorCurrentPriority,
	PrimaryNetworkValidatorPendingPriority:          PrimaryNetworkValidatorCurrentPriority,
	PrimaryNetworkDelegatorBanffPendingPriority:     PrimaryNetworkDelegatorCurrentPriority,
	ChainPermissionlessValidatorPendingPriority:     ChainPermissionlessValidatorCurrentPriority,
	ChainPermissionlessDelegatorPendingPriority:     ChainPermissionlessDelegatorCurrentPriority,
	ChainPermissionedValidatorPendingPriority:       ChainPermissionedValidatorCurrentPriority,
}

type Priority byte

func (p Priority) IsCurrent() bool {
	return p.IsCurrentValidator() || p.IsCurrentDelegator()
}

func (p Priority) IsPending() bool {
	return p.IsPendingValidator() || p.IsPendingDelegator()
}

func (p Priority) IsValidator() bool {
	return p.IsCurrentValidator() || p.IsPendingValidator()
}

func (p Priority) IsPermissionedValidator() bool {
	return p == ChainPermissionedValidatorCurrentPriority ||
		p == ChainPermissionedValidatorPendingPriority
}

func (p Priority) IsDelegator() bool {
	return p.IsCurrentDelegator() || p.IsPendingDelegator()
}

func (p Priority) IsCurrentValidator() bool {
	return p == PrimaryNetworkValidatorCurrentPriority ||
		p == ChainPermissionedValidatorCurrentPriority ||
		p == ChainPermissionlessValidatorCurrentPriority
}

func (p Priority) IsCurrentDelegator() bool {
	return p == PrimaryNetworkDelegatorCurrentPriority ||
		p == ChainPermissionlessDelegatorCurrentPriority
}

func (p Priority) IsPendingValidator() bool {
	return p == PrimaryNetworkValidatorPendingPriority ||
		p == ChainPermissionedValidatorPendingPriority ||
		p == ChainPermissionlessValidatorPendingPriority
}

func (p Priority) IsPendingDelegator() bool {
	return p == PrimaryNetworkDelegatorBanffPendingPriority ||
		p == PrimaryNetworkDelegatorApricotPendingPriority ||
		p == ChainPermissionlessDelegatorPendingPriority
}
