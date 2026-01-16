// Copyright (C) 2022, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

// Package constants provides SDK-specific constants.
// Many constants are re-exported from github.com/luxfi/constants for convenience.
package constants

import (
	"time"

	luxconstants "github.com/luxfi/constants"
)

// Re-export commonly used constants from luxfi/constants
const (
	// Directory structure
	BaseDirName      = luxconstants.BaseDirName
	NodeBinaryName   = luxconstants.NodeBinaryName
	BinDir           = luxconstants.BinDir
	NetDir           = luxconstants.NetDir
	SnapshotsDir     = luxconstants.SnapshotsDir
	SnapshotsDirName = luxconstants.SnapshotsDirName
	RunsDir          = luxconstants.RunsDir
	RunDir           = luxconstants.RunDir
	PluginsDir       = luxconstants.PluginsDir
	PluginDir        = luxconstants.PluginDir
	LogDir           = luxconstants.LogDir
	ConfigDir        = luxconstants.ConfigDir
	KeyDir           = luxconstants.KeyDir
	ChainsDir        = luxconstants.ChainsDir
	CustomVMDir      = luxconstants.CustomVMDir
	ReposDir         = luxconstants.ReposDir
	LPMDir           = luxconstants.LPMDir
	LPMPluginDir     = luxconstants.LPMPluginDir
	DevDir           = luxconstants.DevDir
	LuxCliBinDir     = luxconstants.LuxCliBinDir

	// File permissions
	DefaultPerms755            = luxconstants.DefaultPerms755
	WriteReadReadPerms         = luxconstants.WriteReadReadPerms
	WriteReadOnlyPerms         = luxconstants.WriteReadOnlyPerms
	UserOnlyWriteReadExecPerms = luxconstants.UserOnlyWriteReadExecPerms

	// File names
	ElasticNetConfigFileName  = luxconstants.ElasticNetConfigFileName
	LPMLogName                = luxconstants.LPMLogName
	UpgradeBytesLockExtension = luxconstants.UpgradeBytesLockExtension

	// API endpoints
	LocalAPIEndpoint   = luxconstants.LocalAPIEndpoint
	TestnetAPIEndpoint = luxconstants.TestnetAPIEndpoint
	MainnetAPIEndpoint = luxconstants.MainnetAPIEndpoint

	// WebSocket endpoints
	LocalWSEndpoint   = luxconstants.LocalWSEndpoint
	TestnetWSEndpoint = luxconstants.TestnetWSEndpoint
	MainnetWSEndpoint = luxconstants.MainnetWSEndpoint

	// Default ports
	DefaultHTTPPort    = luxconstants.DefaultHTTPPort
	DefaultStakingPort = luxconstants.DefaultStakingPort
)

// Timeouts - SDK and VM specific
const (
	APIRequestTimeout      = 30 * time.Second
	APIRequestLargeTimeout = 5 * time.Minute
	RequestTimeout         = 3 * time.Minute
)

// Staking file names
const (
	StakerCertFileName = "staker.crt"
	StakerKeyFileName  = "staker.key"
	BLSKeyFileName     = "signer.key"
)

// SSH configuration for remote deployment
const (
	AnsibleSSHShellParams = "-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
	RemoteSSHUser         = "ubuntu"
)

// Snapshot constants
const (
	SnapshotPrefix      = luxconstants.SnapshotPrefix
	DefaultSnapshotName = luxconstants.DefaultSnapshotName
)
