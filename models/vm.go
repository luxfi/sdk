// Copyright (C) 2022, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.
package models

import "github.com/luxfi/constants"

type VMType string

const (
	EVM         = "Lux EVM"
	BlobVM      = "Blob VM"
	TimestampVM = "Timestamp VM"
	SessionVM   = "Session VM"
	ParsVM      = "Pars VM" // Pars network (EVM + SessionVM)
	CustomVM    = "Custom"
)

func VMTypeFromString(s string) VMType {
	switch s {
	case EVM:
		return EVM
	case BlobVM:
		return BlobVM
	case TimestampVM:
		return TimestampVM
	case SessionVM:
		return SessionVM
	case ParsVM:
		return ParsVM
	default:
		return CustomVM
	}
}

func (v VMType) RepoName() string {
	switch v {
	case EVM:
		return constants.EVMRepoName
	case SessionVM:
		return "session" // github.com/luxfi/session
	case ParsVM:
		return "node" // github.com/parsdao/node
	default:
		return "unknown"
	}
}

// Org returns the GitHub organization for the VM type.
func (v VMType) Org() string {
	switch v {
	case ParsVM:
		return "parsdao"
	case SessionVM:
		return constants.LuxOrg // luxfi/session
	default:
		return constants.LuxOrg
	}
}
