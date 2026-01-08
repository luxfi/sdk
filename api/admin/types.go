// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package admin

import (
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
)

// AliasArgs are the arguments for calling Alias.
type AliasArgs struct {
	Endpoint string `json:"endpoint"`
	Alias    string `json:"alias"`
}

// AliasChainArgs are the arguments for calling AliasChain.
type AliasChainArgs struct {
	Chain string `json:"chain"`
	Alias string `json:"alias"`
}

// GetChainAliasesArgs are the arguments for calling GetChainAliases.
type GetChainAliasesArgs struct {
	Chain string `json:"chain"`
}

// GetChainAliasesReply are the aliases of the given chain.
type GetChainAliasesReply struct {
	Aliases []string `json:"aliases"`
}

// SetLoggerLevelArgs are the arguments for calling SetLoggerLevel.
type SetLoggerLevelArgs struct {
	LoggerName   string     `json:"loggerName"`
	LogLevel     *log.Level `json:"logLevel"`
	DisplayLevel *log.Level `json:"displayLevel"`
}

// LogAndDisplayLevels holds log and display levels for a logger.
type LogAndDisplayLevels struct {
	LogLevel     log.Level `json:"logLevel"`
	DisplayLevel log.Level `json:"displayLevel"`
}

// LoggerLevelReply is the reply for log level calls.
type LoggerLevelReply struct {
	LoggerLevels map[string]LogAndDisplayLevels `json:"loggerLevels"`
}

// GetLoggerLevelArgs are the arguments for calling GetLoggerLevel.
type GetLoggerLevelArgs struct {
	LoggerName string `json:"loggerName"`
}

// LoadVMsReply is the response from loading VMs.
type LoadVMsReply struct {
	NewVMs         map[ids.ID][]string `json:"newVMs"`
	FailedVMs      map[ids.ID]string   `json:"failedVMs,omitempty"`
	ChainsRetried int `json:"chainsRetried,omitempty"`
}

// DBGetArgs are the arguments for dbGet.
type DBGetArgs struct {
	Key string `json:"key"`
}

// DBGetReply is the reply for dbGet.
type DBGetReply struct {
	Value string `json:"value"`
}

// VMInfo describes a VM.
type VMInfo struct {
	ID      string   `json:"id"`
	Aliases []string `json:"aliases"`
	Path    string   `json:"path,omitempty"`
}

// ListVMsReply is the reply for listVMs.
type ListVMsReply struct {
	VMs map[string]VMInfo `json:"vms"`
}

// SetTrackedChainsArgs are the arguments for setting tracked chains.
type SetTrackedChainsArgs struct {
	Chains []string `json:"chains"`
}

// SetTrackedChainsReply is the reply for setting tracked chains.
type SetTrackedChainsReply struct {
	TrackedChains []string `json:"trackedChains"`
}

// GetTrackedChainsReply is the reply for getting tracked chains.
type GetTrackedChainsReply struct {
	TrackedChains []string `json:"trackedChains"`
}

// SnapshotArgs are the arguments for snapshot.
type SnapshotArgs struct {
	Path  string `json:"path"`
	Since uint64 `json:"since"`
}

// SnapshotReply is the response from snapshot.
type SnapshotReply struct {
	Success bool   `json:"success"`
	Version uint64 `json:"version"`
}

// LoadArgs are the arguments for load.
type LoadArgs struct {
	Path string `json:"path"`
}
