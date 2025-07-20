// (c) 2025, Hanzo Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package customlogs

import ethtypes "github.com/luxfi/geth/core/types"

// FlattenLogs converts a nested array of logs to a single array of logs.
func FlattenLogs(list [][]*ethtypes.Log) []*ethtypes.Log {
	var flat []*ethtypes.Log
	for _, logs := range list {
		flat = append(flat, logs...)
	}
	return flat
}
