// Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

type Builder interface {
	Run()
	TriggerBuild()
	HandleGenerateBlock()
	Done() // wait after stop
}
