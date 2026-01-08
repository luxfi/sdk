// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package admin

import sdkadmin "github.com/luxfi/sdk/api/admin"

// Re-export shared API types from sdk/api/admin.

type AliasArgs = sdkadmin.AliasArgs

type AliasChainArgs = sdkadmin.AliasChainArgs

type GetChainAliasesArgs = sdkadmin.GetChainAliasesArgs

type GetChainAliasesReply = sdkadmin.GetChainAliasesReply

type SetLoggerLevelArgs = sdkadmin.SetLoggerLevelArgs

type LogAndDisplayLevels = sdkadmin.LogAndDisplayLevels

type LoggerLevelReply = sdkadmin.LoggerLevelReply

type GetLoggerLevelArgs = sdkadmin.GetLoggerLevelArgs

type LoadVMsReply = sdkadmin.LoadVMsReply

type DBGetArgs = sdkadmin.DBGetArgs

type DBGetReply = sdkadmin.DBGetReply

type VMInfo = sdkadmin.VMInfo

type ListVMsReply = sdkadmin.ListVMsReply

type SetTrackedChainsArgs = sdkadmin.SetTrackedChainsArgs

type SetTrackedChainsReply = sdkadmin.SetTrackedChainsReply

type GetTrackedChainsReply = sdkadmin.GetTrackedChainsReply

type SnapshotArgs = sdkadmin.SnapshotArgs

type SnapshotReply = sdkadmin.SnapshotReply

type LoadArgs = sdkadmin.LoadArgs
