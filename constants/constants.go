// Copyright (C) 2022, Lux Industries Inc All rights reserved.
// See the file LICENSE for licensing terms.

// Package constants provides SDK-specific constants.
// Many constants are re-exported from github.com/luxfi/constants for convenience.
package constants

import (
	"strings"
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

// Base is the prefix every luxd route hangs off. There is one -- /v1/info,
// /v1/health, /v1/chain/P -- and the /ext prefix it replaced is gone from the
// node entirely, so an address built any other way is a 404.
const Base = "/v1"

// Route is the address of one of a node's APIs on the node at uri.
//
// It is the ONE place this SDK composes a node address. Before it there were
// ten, every one of them carrying the /ext prefix the node stopped serving,
// which is the shape this drift always takes: an address is a string, a string
// is cheap to repeat, and each copy is somewhere to be wrong on its own.
func Route(uri string, parts ...string) string {
	var b strings.Builder
	b.WriteString(strings.TrimSuffix(uri, "/"))
	b.WriteString(Base)
	for _, p := range parts {
		b.WriteByte('/')
		b.WriteString(strings.Trim(p, "/"))
	}
	return b.String()
}

// Chain is the address of one chain's API -- /v1/chain/P, /v1/chain/X.
//
// The segment is luxconstants.ChainAliasPrefix and not a literal because it
// MOVED: it was "bc" and is "chain", and the node deleted "bc" rather than
// serving both. Reading the constant is what carried this SDK across that
// move; a copy of the old spelling would just be a 404 spelled out longhand.
func Chain(uri, alias string) string {
	return Route(uri, luxconstants.ChainAliasPrefix, alias)
}

// VM is the address of a VM's static API: the handlers a VM serves before any
// chain of it exists, so they are keyed by the VM and not by a chain.
func VM(uri, name string) string {
	return Route(uri, luxconstants.VMAliasPrefix, name)
}

// Index is the address of one chain's index API -- /v1/index/C/block.
func Index(uri, chain, what string) string { return Route(uri, "index", chain, what) }
