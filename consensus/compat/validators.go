package compat

import (
	"context"
	"fmt"

	"github.com/luxfi/ids"
)

// ValidatorConnector handles validator connections
type ValidatorConnector interface {
	// Connected is called when a validator connects
	Connected(ctx context.Context, nodeID ids.NodeID, nodeVersion *Application) error
	// Disconnected is called when a validator disconnects
	Disconnected(ctx context.Context, nodeID ids.NodeID) error
}

// Application version information
type Application struct {
	Name  string
	Major int
	Minor int
	Patch int
}

// String returns the string representation of the version
func (v *Application) String() string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%s/%d.%d.%d", v.Name, v.Major, v.Minor, v.Patch)
}

// Compare returns:
// -1 if v < o
// 0 if v == o
// 1 if v > o
func (v *Application) Compare(o *Application) int {
	if v == nil || o == nil {
		if v == o {
			return 0
		}
		if v == nil {
			return -1
		}
		return 1
	}

	if v.Major != o.Major {
		if v.Major < o.Major {
			return -1
		}
		return 1
	}
	if v.Minor != o.Minor {
		if v.Minor < o.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != o.Patch {
		if v.Patch < o.Patch {
			return -1
		}
		return 1
	}
	return 0
}
