// (c) 2024, Lux Industries, Inc.
//
// This file is a derived work, based on the go-ethereum library whose original
// notices appear below.
//
// It is distributed under a license compatible with the licensing terms of the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********
// Copyright 2022 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import (
	"github.com/ethereum/go-ethereum/metrics"
)

// ====== If resolving merge conflicts ======
//
// All calls to metrics.NewRegistered*() for metrics also defined in libevm/triedb/pathdb
// have been replaced with metrics.GetOrRegister*() to get metrics already registered in
// libevm/triedb/pathdb or register them here otherwise. These replacements ensure the same
// metrics are shared between the two packages.
//
//nolint:unused
var (
	cleanHitMeter   = metrics.NewMeter()
	cleanMissMeter  = metrics.NewMeter()
	cleanReadMeter  = metrics.NewMeter()
	cleanWriteMeter = metrics.NewMeter()

	dirtyHitMeter         = metrics.NewMeter()
	dirtyMissMeter        = metrics.NewMeter()
	dirtyReadMeter        = metrics.NewMeter()
	dirtyWriteMeter       = metrics.NewMeter()
	dirtyNodeHitDepthHist = metrics.NewHistogram(metrics.NewExpDecaySample(1028, 0.015))

	cleanFalseMeter = metrics.NewMeter()
	dirtyFalseMeter = metrics.NewMeter()
	diskFalseMeter  = metrics.NewMeter()

	commitTimeTimer  = metrics.NewTimer()
	commitNodesMeter = metrics.NewMeter()
	commitBytesMeter = metrics.NewMeter()

	gcNodesMeter = metrics.NewMeter()
	gcBytesMeter = metrics.NewMeter()

	diffLayerBytesMeter = metrics.GetOrRegisterMeter("pathdb/diff/bytes", nil)
	diffLayerNodesMeter = metrics.GetOrRegisterMeter("pathdb/diff/nodes", nil)

	historyBuildTimeMeter  = metrics.GetOrRegisterTimer("pathdb/history/time", nil)
	historyDataBytesMeter  = metrics.GetOrRegisterMeter("pathdb/history/bytes/data", nil)
	historyIndexBytesMeter = metrics.GetOrRegisterMeter("pathdb/history/bytes/index", nil)
)
