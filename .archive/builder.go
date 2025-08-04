// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

type Builder interface {
	Run()
	TriggerBuild()
	HandleGenerateBlock()
	Done() // wait after stop
}
