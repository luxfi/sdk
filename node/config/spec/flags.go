// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package spec

import (
	"runtime"
	"time"
)

// allFlags returns all flag specifications.
// This is generated from the flag definitions in config/flags.go
func allFlags() []FlagSpec {
	return []FlagSpec{
		// ============================================================================
		// Process Flags
		// ============================================================================
		{
			Key:         "version",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, print version and quit",
			Category:    CategoryProcess,
		},
		{
			Key:         "version-json",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, print version in JSON format and quit",
			Category:    CategoryProcess,
		},

		// ============================================================================
		// Node Flags
		// ============================================================================
		{
			Key:         "dev",
			Type:        TypeBool,
			Default:     false,
			Description: "Enables development mode with single-node consensus, no sybil protection, and other dev-friendly settings",
			Category:    CategoryDev,
		},
		{
			Key:         "data-dir",
			Type:        TypeString,
			Default:     "$HOME/.lux",
			Description: "Sets the base data directory where default sub-directories will be placed unless otherwise specified",
			Category:    CategoryNode,
		},
		{
			Key:         "fd-limit",
			Type:        TypeUint64,
			Default:     uint64(32768),
			Description: "Attempts to raise the process file descriptor limit to at least this value and error if the value is above the system max",
			Category:    CategoryNode,
		},
		{
			Key:         "plugin-dir",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/plugins/current",
			Description: "Path to the plugin directory",
			Category:    CategoryNode,
		},
		{
			Key:         "config-file",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies a config file. Ignored if config-file-content is specified",
			Category:    CategoryNode,
		},
		{
			Key:         "config-file-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded config content",
			Category:    CategoryNode,
			Sensitive:   true,
		},
		{
			Key:         "config-file-content-type",
			Type:        TypeString,
			Default:     "json",
			Description: "Specifies the format of the base64 encoded config content. Available values: 'json', 'yaml', 'toml'",
			Category:    CategoryNode,
			Constraints: &Constraints{
				Enum: []string{"json", "yaml", "toml"},
			},
		},

		// ============================================================================
		// Genesis Flags
		// ============================================================================
		{
			Key:         "genesis-file",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies a genesis config file path. Ignored when running standard networks or if genesis-file-content is specified",
			Category:    CategoryGenesis,
		},
		{
			Key:         "genesis-file-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded genesis content",
			Category:    CategoryGenesis,
		},
		{
			Key:         "genesis-db",
			Type:        TypeString,
			Default:     "",
			Description: "Path to existing genesis database for replay. Cannot be used with genesis-file or genesis-file-content",
			Category:    CategoryGenesis,
			Constraints: &Constraints{
				ConflictsWith: []string{"genesis-file", "genesis-file-content"},
			},
		},
		{
			Key:         "genesis-db-type",
			Type:        TypeString,
			Default:     "badgerdb",
			Description: "Database type to use for genesis database. Must be one of {pebbledb, badgerdb}",
			Category:    CategoryGenesis,
			Constraints: &Constraints{
				Enum: []string{"pebbledb", "badgerdb"},
			},
		},
		{
			Key:         "genesis-block-limit",
			Type:        TypeUint64,
			Default:     uint64(0),
			Description: "Limit number of blocks to replay during genesis (0 = all blocks)",
			Category:    CategoryGenesis,
		},
		{
			Key:         "allow-custom-genesis",
			Type:        TypeBool,
			Default:     true,
			Description: "Allow custom genesis for mainnet/testnet (default: true for development, set false for production)",
			Category:    CategoryGenesis,
		},
		{
			Key:         "upgrade-file",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies an upgrade config file path. Ignored when running standard networks or if upgrade-file-content is specified",
			Category:    CategoryGenesis,
		},
		{
			Key:         "upgrade-file-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded upgrade content",
			Category:    CategoryGenesis,
		},
		{
			Key:         "network-id",
			Type:        TypeString,
			Default:     "mainnet",
			Description: "Network ID this node will connect to",
			Category:    CategoryNetwork,
		},

		// ============================================================================
		// LP Flags
		// ============================================================================
		{
			Key:         "lp-support",
			Type:        TypeIntSlice,
			Default:     nil,
			Description: "LPs to support adoption",
			Category:    CategoryDev,
		},
		{
			Key:         "lp-object",
			Type:        TypeIntSlice,
			Default:     nil,
			Description: "LPs to object adoption",
			Category:    CategoryDev,
		},

		// ============================================================================
		// Fee Flags
		// ============================================================================
		{
			Key:         "validator-fees-capacity",
			Type:        TypeUint64,
			Default:     uint64(20000),
			Description: "Maximum number of validators",
			Category:    CategoryFees,
		},
		{
			Key:         "validator-fees-target",
			Type:        TypeUint64,
			Default:     uint64(10000),
			Description: "Target number of validators",
			Category:    CategoryFees,
		},
		{
			Key:         "validator-fees-min-price",
			Type:        TypeUint64,
			Default:     uint64(512),
			Description: "Minimum validator price in nLUX per second",
			Category:    CategoryFees,
		},
		{
			Key:         "validator-fees-excess-conversion-constant",
			Type:        TypeUint64,
			Default:     uint64(107411168183),
			Description: "Constant to convert validator excess price",
			Category:    CategoryFees,
		},
		{
			Key:         "dynamic-fees-bandwidth-weight",
			Type:        TypeUint64,
			Default:     uint64(1),
			Description: "Complexity multiplier used to convert Bandwidth into Gas",
			Category:    CategoryFees,
		},
		{
			Key:         "dynamic-fees-db-read-weight",
			Type:        TypeUint64,
			Default:     uint64(1),
			Description: "Complexity multiplier used to convert DB Reads into Gas",
			Category:    CategoryFees,
		},
		{
			Key:         "dynamic-fees-db-write-weight",
			Type:        TypeUint64,
			Default:     uint64(1),
			Description: "Complexity multiplier used to convert DB Writes into Gas",
			Category:    CategoryFees,
		},
		{
			Key:         "dynamic-fees-compute-weight",
			Type:        TypeUint64,
			Default:     uint64(1),
			Description: "Complexity multiplier used to convert Compute into Gas",
			Category:    CategoryFees,
		},
		{
			Key:         "dynamic-fees-max-gas-capacity",
			Type:        TypeUint64,
			Default:     uint64(1000000),
			Description: "Maximum amount of Gas the chain is allowed to store for future use",
			Category:    CategoryFees,
		},
		{
			Key:         "dynamic-fees-max-gas-per-second",
			Type:        TypeUint64,
			Default:     uint64(1000),
			Description: "Rate at which Gas is stored for future use",
			Category:    CategoryFees,
		},
		{
			Key:         "dynamic-fees-target-gas-per-second",
			Type:        TypeUint64,
			Default:     uint64(500),
			Description: "Target rate of Gas usage",
			Category:    CategoryFees,
		},
		{
			Key:         "dynamic-fees-min-gas-price",
			Type:        TypeUint64,
			Default:     uint64(1),
			Description: "Minimum Gas price",
			Category:    CategoryFees,
		},
		{
			Key:         "dynamic-fees-excess-conversion-constant",
			Type:        TypeUint64,
			Default:     uint64(5761920000000),
			Description: "Constant to convert excess Gas to the Gas price",
			Category:    CategoryFees,
		},
		{
			Key:         "tx-fee",
			Type:        TypeUint64,
			Default:     uint64(1000000),
			Description: "Transaction fee, in nLUX",
			Category:    CategoryFees,
		},
		{
			Key:         "create-asset-tx-fee",
			Type:        TypeUint64,
			Default:     uint64(10000000),
			Description: "Transaction fee, in nLUX, for transactions that create new assets",
			Category:    CategoryFees,
		},

		// ============================================================================
		// Database Flags
		// ============================================================================
		{
			Key:         "db-type",
			Type:        TypeString,
			Default:     "badgerdb",
			Description: "Default database type to use for all chains. Must be one of {badgerdb, pebbledb, memdb}",
			Category:    CategoryDatabase,
			Constraints: &Constraints{
				Enum: []string{"badgerdb", "pebbledb", "memdb"},
			},
		},
		{
			Key:         "db-read-only",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, database writes are to memory and never persisted. May still initialize database directory/files on disk if they don't exist",
			Category:    CategoryDatabase,
		},
		{
			Key:         "db-dir",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/db",
			Description: "Path to database directory",
			Category:    CategoryDatabase,
		},
		{
			Key:         "db-config-file",
			Type:        TypeString,
			Default:     "",
			Description: "Path to database config file. Ignored if db-config-file-content is specified",
			Category:    CategoryDatabase,
		},
		{
			Key:         "db-config-file-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded database config content",
			Category:    CategoryDatabase,
		},
		{
			Key:         "p-chain-db-type",
			Type:        TypeString,
			Default:     "",
			Description: "Database type for P-Chain. If not specified, uses default db-type",
			Category:    CategoryDatabase,
		},
		{
			Key:         "x-chain-db-type",
			Type:        TypeString,
			Default:     "",
			Description: "Database type for X-Chain. If not specified, uses default db-type",
			Category:    CategoryDatabase,
		},
		{
			Key:         "c-chain-db-type",
			Type:        TypeString,
			Default:     "",
			Description: "Database type for C-Chain. If not specified, uses default db-type",
			Category:    CategoryDatabase,
		},

		// ============================================================================
		// Logging Flags
		// ============================================================================
		{
			Key:         "log-dir",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/logs",
			Description: "Logging directory for Lux",
			Category:    CategoryLogging,
		},
		{
			Key:         "log-level",
			Type:        TypeString,
			Default:     "info",
			Description: "The log level. Should be one of {verbo, debug, trace, info, warn, error, fatal, off}",
			Category:    CategoryLogging,
			Constraints: &Constraints{
				Enum: []string{"verbo", "debug", "trace", "info", "warn", "error", "fatal", "off"},
			},
		},
		{
			Key:         "log-display-level",
			Type:        TypeString,
			Default:     "",
			Description: "The log display level. If left blank, will inherit the value of log-level. Otherwise, should be one of {verbo, debug, trace, info, warn, error, fatal, off}",
			Category:    CategoryLogging,
		},
		{
			Key:         "log-rotater-max-size",
			Type:        TypeUint,
			Default:     uint(8),
			Description: "The maximum file size in megabytes of the log file before it gets rotated",
			Category:    CategoryLogging,
		},
		{
			Key:         "log-rotater-max-files",
			Type:        TypeUint,
			Default:     uint(7),
			Description: "The maximum number of old log files to retain. 0 means retain all old log files",
			Category:    CategoryLogging,
		},
		{
			Key:         "log-rotater-max-age",
			Type:        TypeUint,
			Default:     uint(0),
			Description: "The maximum number of days to retain old log files based on the timestamp encoded in their filename. 0 means retain all old log files",
			Category:    CategoryLogging,
		},
		{
			Key:         "log-rotater-compress-enabled",
			Type:        TypeBool,
			Default:     false,
			Description: "Enables the compression of rotated log files through gzip",
			Category:    CategoryLogging,
		},
		{
			Key:         "log-disable-display-plugin-logs",
			Type:        TypeBool,
			Default:     false,
			Description: "Disables displaying plugin logs in stdout",
			Category:    CategoryLogging,
		},

		// ============================================================================
		// Network Peer List Flags
		// ============================================================================
		{
			Key:         "network-peer-list-num-validator-ips",
			Type:        TypeUint,
			Default:     uint(15),
			Description: "Number of validator IPs to gossip to other nodes",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-peer-list-pull-gossip-frequency",
			Type:        TypeDuration,
			Default:     time.Second,
			Description: "Frequency to request peers from other nodes",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-peer-list-bloom-reset-frequency",
			Type:        TypeDuration,
			Default:     time.Minute,
			Description: "Frequency to recalculate the bloom filter used to request new peers from other nodes",
			Category:    CategoryNetwork,
		},

		// ============================================================================
		// Public IP Flags
		// ============================================================================
		{
			Key:         "public-ip",
			Type:        TypeString,
			Default:     "",
			Description: "Public IP of this node for P2P communication",
			Category:    CategoryNetwork,
		},
		{
			Key:         "public-ip-resolution-frequency",
			Type:        TypeDuration,
			Default:     5 * time.Minute,
			Description: "Frequency at which this node resolves/updates its public IP and renew NAT mappings, if applicable",
			Category:    CategoryNetwork,
		},
		{
			Key:         "public-ip-resolution-service",
			Type:        TypeString,
			Default:     "",
			Description: "Only acceptable values are 'opendns', 'ifconfigco' or 'ifconfigme'. When provided, the node will use that service to periodically resolve/update its public IP",
			Category:    CategoryNetwork,
			Constraints: &Constraints{
				Enum: []string{"", "opendns", "ifconfigco", "ifconfigme"},
			},
		},

		// ============================================================================
		// Network Throttling Flags
		// ============================================================================
		{
			Key:         "network-inbound-connection-throttling-cooldown",
			Type:        TypeDuration,
			Default:     10 * time.Second,
			Description: "Upgrade an inbound connection from a given IP at most once per this duration. If 0, don't rate-limit inbound connection upgrades",
			Category:    CategoryThrottler,
		},
		{
			Key:         "network-inbound-connection-throttling-max-conns-per-sec",
			Type:        TypeFloat64,
			Default:     256.0,
			Description: "Max number of inbound connections to accept (from all peers) per second",
			Category:    CategoryThrottler,
		},
		{
			Key:         "network-outbound-connection-throttling-rps",
			Type:        TypeUint,
			Default:     uint(50),
			Description: "Make at most this number of outgoing peer connection attempts per second",
			Category:    CategoryThrottler,
		},
		{
			Key:         "network-outbound-connection-timeout",
			Type:        TypeDuration,
			Default:     30 * time.Second,
			Description: "Timeout when dialing a peer",
			Category:    CategoryNetwork,
		},

		// ============================================================================
		// Network Timeout Flags
		// ============================================================================
		{
			Key:         "network-initial-timeout",
			Type:        TypeDuration,
			Default:     5 * time.Second,
			Description: "Initial timeout value of the adaptive timeout manager",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-minimum-timeout",
			Type:        TypeDuration,
			Default:     2 * time.Second,
			Description: "Minimum timeout value of the adaptive timeout manager",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-maximum-timeout",
			Type:        TypeDuration,
			Default:     10 * time.Second,
			Description: "Maximum timeout value of the adaptive timeout manager",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-maximum-inbound-timeout",
			Type:        TypeDuration,
			Default:     10 * time.Second,
			Description: "Maximum timeout value of an inbound message. Defines duration within which an incoming message must be fulfilled. Incoming messages containing deadline higher than this value will be overridden with this value",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-timeout-halflife",
			Type:        TypeDuration,
			Default:     5 * time.Minute,
			Description: "Halflife of average network response time. Higher value --> network timeout is less volatile. Can't be 0",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-timeout-coefficient",
			Type:        TypeFloat64,
			Default:     2.0,
			Description: "Multiplied by average network response time to get the network timeout. Must be >= 1",
			Category:    CategoryNetwork,
			Constraints: &Constraints{
				Min: 1.0,
			},
		},
		{
			Key:         "network-read-handshake-timeout",
			Type:        TypeDuration,
			Default:     15 * time.Second,
			Description: "Timeout value for reading handshake messages",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-ping-timeout",
			Type:        TypeDuration,
			Default:     30 * time.Second,
			Description: "Timeout value for Ping-Pong with a peer",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-ping-frequency",
			Type:        TypeDuration,
			Default:     22500 * time.Millisecond,
			Description: "Frequency of pinging other peers",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-no-ingress-connections-grace-period",
			Type:        TypeDuration,
			Default:     time.Minute,
			Description: "Time after which nodes are expected to be connected to us if we are a primary network validator, otherwise a health check fails",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-compression-type",
			Type:        TypeString,
			Default:     "zstd",
			Description: "Compression type for outbound messages. Must be one of [zstd, none]",
			Category:    CategoryNetwork,
			Constraints: &Constraints{
				Enum: []string{"zstd", "none"},
			},
		},
		{
			Key:         "network-max-clock-difference",
			Type:        TypeDuration,
			Default:     time.Minute,
			Description: "Max allowed clock difference value between this node and peers",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-allow-private-ips",
			Type:        TypeBool,
			Default:     true,
			Description: "Allows the node to initiate outbound connection attempts to peers with private IPs. Default is true. Set to false to prohibit for security-sensitive production deployments",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-require-validator-to-connect",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, this node will only maintain a connection with another node if this node is a validator, the other node is a validator, or the other node is a beacon",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-peer-read-buffer-size",
			Type:        TypeUint,
			Default:     uint(8 * 1024),
			Description: "Size, in bytes, of the buffer that we read peer messages into (there is one buffer per peer)",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-peer-write-buffer-size",
			Type:        TypeUint,
			Default:     uint(8 * 1024),
			Description: "Size, in bytes, of the buffer that we write peer messages into (there is one buffer per peer)",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-tcp-proxy-enabled",
			Type:        TypeBool,
			Default:     false,
			Description: "Require all P2P connections to be initiated with a TCP proxy header",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-tcp-proxy-read-timeout",
			Type:        TypeDuration,
			Default:     3 * time.Second,
			Description: "Maximum duration to wait for a TCP proxy header",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-tls-key-log-file-unsafe",
			Type:        TypeString,
			Default:     "",
			Description: "TLS key log file path. Should only be specified for debugging",
			Category:    CategoryNetwork,
			Sensitive:   true,
		},

		// ============================================================================
		// Benchlist Flags
		// ============================================================================
		{
			Key:         "benchlist-fail-threshold",
			Type:        TypeInt,
			Default:     10,
			Description: "Number of consecutive failed queries before benchlisting a node",
			Category:    CategoryNetwork,
		},
		{
			Key:         "benchlist-duration",
			Type:        TypeDuration,
			Default:     15 * time.Minute,
			Description: "Max amount of time a peer is benchlisted after surpassing the threshold",
			Category:    CategoryNetwork,
		},
		{
			Key:         "benchlist-min-failing-duration",
			Type:        TypeDuration,
			Default:     2*time.Minute + 30*time.Second,
			Description: "Minimum amount of time messages to a peer must be failing before the peer is benched",
			Category:    CategoryNetwork,
		},

		// ============================================================================
		// Router/Consensus Flags
		// ============================================================================
		{
			Key:         "consensus-app-concurrency",
			Type:        TypeUint,
			Default:     uint(2),
			Description: "Maximum number of goroutines to use when handling App messages on a chain",
			Category:    CategoryConsensus,
		},
		{
			Key:         "consensus-shutdown-timeout",
			Type:        TypeDuration,
			Default:     30 * time.Second,
			Description: "Timeout before killing an unresponsive chain",
			Category:    CategoryConsensus,
		},
		{
			Key:         "consensus-frontier-poll-frequency",
			Type:        TypeDuration,
			Default:     100 * time.Millisecond,
			Description: "Frequency of polling for new consensus frontiers",
			Category:    CategoryConsensus,
		},

		// ============================================================================
		// Inbound Throttler Flags
		// ============================================================================
		{
			Key:         "throttler-inbound-at-large-alloc-size",
			Type:        TypeUint64,
			Default:     uint64(6 * 1024 * 1024),
			Description: "Size, in bytes, of at-large byte allocation in inbound message throttler",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-inbound-validator-alloc-size",
			Type:        TypeUint64,
			Default:     uint64(32 * 1024 * 1024),
			Description: "Size, in bytes, of validator byte allocation in inbound message throttler",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-inbound-node-max-at-large-bytes",
			Type:        TypeUint64,
			Default:     uint64(2 * 1024 * 1024),
			Description: "Max number of bytes a node can take from the inbound message throttler's at-large allocation. Must be at least the max message size",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-inbound-node-max-processing-msgs",
			Type:        TypeUint64,
			Default:     uint64(1024),
			Description: "Max number of messages currently processing from a given node",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-inbound-bandwidth-refill-rate",
			Type:        TypeUint64,
			Default:     uint64(512 * 1024),
			Description: "Max average inbound bandwidth usage of a peer, in bytes per second. See BandwidthThrottler",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-inbound-bandwidth-max-burst-size",
			Type:        TypeUint64,
			Default:     uint64(2 * 1024 * 1024),
			Description: "Max inbound bandwidth a node can use at once. Must be at least the max message size. See BandwidthThrottler",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-inbound-cpu-max-recheck-delay",
			Type:        TypeDuration,
			Default:     5 * time.Second,
			Description: "In the CPU-based network throttler, check at least this often whether the node's CPU usage has fallen to an acceptable level",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-inbound-disk-max-recheck-delay",
			Type:        TypeDuration,
			Default:     5 * time.Second,
			Description: "In the disk-based network throttler, check at least this often whether the node's disk usage has fallen to an acceptable level",
			Category:    CategoryThrottler,
		},

		// ============================================================================
		// Outbound Throttler Flags
		// ============================================================================
		{
			Key:         "throttler-outbound-at-large-alloc-size",
			Type:        TypeUint64,
			Default:     uint64(6 * 1024 * 1024),
			Description: "Size, in bytes, of at-large byte allocation in outbound message throttler",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-outbound-validator-alloc-size",
			Type:        TypeUint64,
			Default:     uint64(32 * 1024 * 1024),
			Description: "Size, in bytes, of validator byte allocation in outbound message throttler",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-outbound-node-max-at-large-bytes",
			Type:        TypeUint64,
			Default:     uint64(2 * 1024 * 1024),
			Description: "Max number of bytes a node can take from the outbound message throttler's at-large allocation. Must be at least the max message size",
			Category:    CategoryThrottler,
		},

		// ============================================================================
		// HTTP Flags
		// ============================================================================
		{
			Key:         "http-host",
			Type:        TypeString,
			Default:     "127.0.0.1",
			Description: "Address of the HTTP server. If the address is empty or a literal unspecified IP address, the server will bind on all available unicast and anycast IP addresses of the local system",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-port",
			Type:        TypeUint,
			Default:     uint(9630),
			Description: "Port of the HTTP server. If the port is 0 a port number is automatically chosen",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-tls-enabled",
			Type:        TypeBool,
			Default:     false,
			Description: "Upgrade the HTTP server to HTTPs",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-tls-key-file",
			Type:        TypeString,
			Default:     "",
			Description: "TLS private key file for the HTTPs server. Ignored if http-tls-key-file-content is specified",
			Category:    CategoryHTTP,
			Sensitive:   true,
		},
		{
			Key:         "http-tls-key-file-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded TLS private key for the HTTPs server",
			Category:    CategoryHTTP,
			Sensitive:   true,
		},
		{
			Key:         "http-tls-cert-file",
			Type:        TypeString,
			Default:     "",
			Description: "TLS certificate file for the HTTPs server. Ignored if http-tls-cert-file-content is specified",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-tls-cert-file-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded TLS certificate for the HTTPs server",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-allowed-origins",
			Type:        TypeString,
			Default:     "*",
			Description: "Origins to allow on the HTTP port. Defaults to * which allows all origins. Example: https://*.lux.network https://*.lux-test.network",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-allowed-hosts",
			Type:        TypeStringSlice,
			Default:     []string{"localhost"},
			Description: "List of acceptable host names in API requests. Provide the wildcard ('*') to accept requests from all hosts. API requests where the Host field is empty or an IP address will always be accepted. An API call whose HTTP Host field isn't acceptable will receive a 403 error code",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-shutdown-wait",
			Type:        TypeDuration,
			Default:     time.Duration(0),
			Description: "Duration to wait after receiving SIGTERM or SIGINT before initiating shutdown. The /health endpoint will return unhealthy during this duration",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-shutdown-timeout",
			Type:        TypeDuration,
			Default:     10 * time.Second,
			Description: "Maximum duration to wait for existing connections to complete during node shutdown",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-read-timeout",
			Type:        TypeDuration,
			Default:     3600 * time.Second,
			Description: "Maximum duration for reading the entire request, including the body. A zero or negative value means there will be no timeout",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-read-header-timeout",
			Type:        TypeDuration,
			Default:     30 * time.Second,
			Description: "Maximum duration to read request headers. The connection's read deadline is reset after reading the headers. If http-read-timeout is zero, the value of http-read-header-timeout is used. If both are zero, there is no timeout",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-write-timeout",
			Type:        TypeDuration,
			Default:     3600 * time.Second,
			Description: "Maximum duration before timing out writes of the response. It is reset whenever a new request's header is read. A zero or negative value means there will be no timeout",
			Category:    CategoryHTTP,
		},
		{
			Key:         "http-idle-timeout",
			Type:        TypeDuration,
			Default:     120 * time.Second,
			Description: "Maximum duration to wait for the next request when keep-alives are enabled. If http-read-timeout is zero, the value of http-idle-timeout is used. If both are zero, there is no timeout",
			Category:    CategoryHTTP,
		},

		// ============================================================================
		// API Flags
		// ============================================================================
		{
			Key:         "api-admin-enabled",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, this node exposes the Admin API",
			Category:    CategoryAPI,
		},
		{
			Key:         "api-info-enabled",
			Type:        TypeBool,
			Default:     true,
			Description: "If true, this node exposes the Info API",
			Category:    CategoryAPI,
		},
		{
			Key:         "api-keystore-enabled",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, this node exposes the Keystore API",
			Category:    CategoryAPI,
		},
		{
			Key:         "api-metrics-enabled",
			Type:        TypeBool,
			Default:     true,
			Description: "If true, this node exposes the Metrics API",
			Category:    CategoryAPI,
		},
		{
			Key:         "api-health-enabled",
			Type:        TypeBool,
			Default:     true,
			Description: "If true, this node exposes the Health API",
			Category:    CategoryAPI,
		},

		// ============================================================================
		// Health Check Flags
		// ============================================================================
		{
			Key:         "health-check-frequency",
			Type:        TypeDuration,
			Default:     30 * time.Second,
			Description: "Time between health checks",
			Category:    CategoryHealth,
		},
		{
			Key:         "health-check-averager-halflife",
			Type:        TypeDuration,
			Default:     10 * time.Second,
			Description: "Halflife of averager when calculating a running average in a health check",
			Category:    CategoryHealth,
		},
		{
			Key:         "network-health-max-time-since-msg-sent",
			Type:        TypeDuration,
			Default:     time.Minute,
			Description: "Network layer returns unhealthy if haven't sent a message for at least this much time",
			Category:    CategoryHealth,
		},
		{
			Key:         "network-health-max-time-since-msg-received",
			Type:        TypeDuration,
			Default:     time.Minute,
			Description: "Network layer returns unhealthy if haven't received a message for at least this much time",
			Category:    CategoryHealth,
		},
		{
			Key:         "network-health-max-portion-send-queue-full",
			Type:        TypeFloat64,
			Default:     0.9,
			Description: "Network layer returns unhealthy if more than this portion of the pending send queue is full",
			Category:    CategoryHealth,
		},
		{
			Key:         "network-health-min-conn-peers",
			Type:        TypeUint,
			Default:     uint(1),
			Description: "Network layer returns unhealthy if connected to less than this many peers",
			Category:    CategoryHealth,
		},
		{
			Key:         "network-health-max-send-fail-rate",
			Type:        TypeFloat64,
			Default:     0.25,
			Description: "Network layer reports unhealthy if more than this portion of attempted message sends fail",
			Category:    CategoryHealth,
		},
		{
			Key:         "router-health-max-drop-rate",
			Type:        TypeFloat64,
			Default:     1.0,
			Description: "Node reports unhealthy if the router drops more than this portion of messages",
			Category:    CategoryHealth,
		},
		{
			Key:         "router-health-max-outstanding-requests",
			Type:        TypeUint,
			Default:     uint(1024),
			Description: "Node reports unhealthy if there are more than this many outstanding consensus requests (Get, PullQuery, etc.) over all chains",
			Category:    CategoryHealth,
		},
		{
			Key:         "network-health-max-outstanding-request-duration",
			Type:        TypeDuration,
			Default:     5 * time.Minute,
			Description: "Node reports unhealthy if there has been a request outstanding for this duration",
			Category:    CategoryHealth,
		},

		// ============================================================================
		// Staking Flags
		// ============================================================================
		{
			Key:         "staking-host",
			Type:        TypeString,
			Default:     "",
			Description: "Address of the consensus server. If the address is empty or a literal unspecified IP address, the server will bind on all available unicast and anycast IP addresses of the local system",
			Category:    CategoryStaking,
		},
		{
			Key:         "staking-port",
			Type:        TypeUint,
			Default:     uint(9631),
			Description: "Port of the consensus server. If the port is 0 a port number is automatically chosen",
			Category:    CategoryStaking,
		},
		{
			Key:         "staking-ephemeral-cert-enabled",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, the node uses an ephemeral staking TLS key and certificate, and has an ephemeral node ID",
			Category:    CategoryStaking,
		},
		{
			Key:         "staking-tls-key-file",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/staking/staker.key",
			Description: "Path to the TLS private key for staking. Ignored if staking-tls-key-file-content is specified",
			Category:    CategoryStaking,
			Sensitive:   true,
		},
		{
			Key:         "staking-tls-key-file-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded TLS private key for staking",
			Category:    CategoryStaking,
			Sensitive:   true,
		},
		{
			Key:         "staking-tls-cert-file",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/staking/staker.crt",
			Description: "Path to the TLS certificate for staking. Ignored if staking-tls-cert-file-content is specified",
			Category:    CategoryStaking,
		},
		{
			Key:         "staking-tls-cert-file-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded TLS certificate for staking",
			Category:    CategoryStaking,
		},
		{
			Key:         "staking-ephemeral-signer-enabled",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, the node uses an ephemeral staking signer key",
			Category:    CategoryStaking,
		},
		{
			Key:         "staking-signer-key-file",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/staking/signer.key",
			Description: "Path to the signer private key for staking. Ignored if staking-signer-key-file-content is specified",
			Category:    CategoryStaking,
			Sensitive:   true,
		},
		{
			Key:         "staking-signer-key-file-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded signer private key for staking",
			Category:    CategoryStaking,
			Sensitive:   true,
		},
		{
			Key:         "sybil-protection-enabled",
			Type:        TypeBool,
			Default:     true,
			Description: "Enables sybil protection. If enabled, Network TLS is required",
			Category:    CategoryStaking,
		},
		{
			Key:         "sybil-protection-disabled-weight",
			Type:        TypeUint64,
			Default:     uint64(100),
			Description: "Weight to provide to each peer when sybil protection is disabled",
			Category:    CategoryStaking,
		},
		{
			Key:         "partial-sync-primary-network",
			Type:        TypeBool,
			Default:     false,
			Description: "Only sync the P-chain on the Primary Network. If the node is a Primary Network validator, it will report unhealthy",
			Category:    CategoryStaking,
		},
		{
			Key:         "uptime-requirement",
			Type:        TypeFloat64,
			Default:     0.8,
			Description: "Fraction of time a validator must be online to receive rewards",
			Category:    CategoryStaking,
		},
		{
			Key:         "min-validator-stake",
			Type:        TypeUint64,
			Default:     uint64(2000000000000),
			Description: "Minimum stake, in nLUX, required to validate the primary network",
			Category:    CategoryStaking,
		},
		{
			Key:         "max-validator-stake",
			Type:        TypeUint64,
			Default:     uint64(3000000000000000),
			Description: "Maximum stake, in nLUX, that can be placed on a validator on the primary network",
			Category:    CategoryStaking,
		},
		{
			Key:         "min-delegator-stake",
			Type:        TypeUint64,
			Default:     uint64(25000000000),
			Description: "Minimum stake, in nLUX, that can be delegated on the primary network",
			Category:    CategoryStaking,
		},
		{
			Key:         "min-delegation-fee",
			Type:        TypeUint64,
			Default:     uint64(20000),
			Description: "Minimum delegation fee, in the range [0, 1000000], that can be charged for delegation on the primary network",
			Category:    CategoryStaking,
		},
		{
			Key:         "min-stake-duration",
			Type:        TypeDuration,
			Default:     24 * time.Hour,
			Description: "Minimum staking duration",
			Category:    CategoryStaking,
		},
		{
			Key:         "max-stake-duration",
			Type:        TypeDuration,
			Default:     365 * 24 * time.Hour,
			Description: "Maximum staking duration",
			Category:    CategoryStaking,
		},
		{
			Key:         "stake-max-consumption-rate",
			Type:        TypeUint64,
			Default:     uint64(120000),
			Description: "Maximum consumption rate of the remaining tokens to mint in the staking function",
			Category:    CategoryStaking,
		},
		{
			Key:         "stake-min-consumption-rate",
			Type:        TypeUint64,
			Default:     uint64(100000),
			Description: "Minimum consumption rate of the remaining tokens to mint in the staking function",
			Category:    CategoryStaking,
		},
		{
			Key:         "stake-minting-period",
			Type:        TypeDuration,
			Default:     365 * 24 * time.Hour,
			Description: "Consumption period of the staking function",
			Category:    CategoryStaking,
		},
		{
			Key:         "stake-supply-cap",
			Type:        TypeUint64,
			Default:     uint64(2000000000000000000),
			Description: "Supply cap of the staking function",
			Category:    CategoryStaking,
		},

		// ============================================================================
		// Chain Tracking Flags
		// ============================================================================
		{
			Key:         "track-chains",
			Type:        TypeString,
			Default:     "",
			Description: "Comma-separated list of chain IDs to track. A node tracking chains will sync those chains and track validator uptimes. Use --track-all-chains for development networks",
			Category:    CategoryChain,
		},
		{
			Key:         "track-all-chains",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, track all chains automatically. Useful for development and testing networks where you want to sync all deployed chains without specifying each one",
			Category:    CategoryChain,
		},

		// ============================================================================
		// State Sync Flags
		// ============================================================================
		{
			Key:         "state-sync-ips",
			Type:        TypeString,
			Default:     "",
			Description: "Comma separated list of state sync peer ips to connect to. Example: 127.0.0.1:9630,127.0.0.1:9631",
			Category:    CategoryBootstrap,
		},
		{
			Key:         "state-sync-ids",
			Type:        TypeString,
			Default:     "",
			Description: "Comma separated list of state sync peer ids to connect to. Example: NodeID-JR4dVmy6ffUGAKCBDkyCbeZbyHQBeDsET,NodeID-8CrVPQZ4VSqgL8zTdvL14G8HqAfrBr4z",
			Category:    CategoryBootstrap,
		},

		// ============================================================================
		// Bootstrap Flags
		// ============================================================================
		{
			Key:         "bootstrap-ips",
			Type:        TypeString,
			Default:     "",
			Description: "Comma separated list of bootstrap peer ips to connect to. Example: 127.0.0.1:9630,127.0.0.1:9631",
			Category:    CategoryBootstrap,
		},
		{
			Key:         "bootstrap-ids",
			Type:        TypeString,
			Default:     "",
			Description: "Comma separated list of bootstrap peer ids to connect to. Example: NodeID-JR4dVmy6ffUGAKCBDkyCbeZbyHQBeDsET,NodeID-8CrVPQZ4VSqgL8zTdvL14G8HqAfrBr4z",
			Category:    CategoryBootstrap,
		},
		{
			Key:         "skip-bootstrap",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, skip the bootstrapping phase and start processing immediately",
			Category:    CategoryBootstrap,
		},
		{
			Key:         "enable-automining",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, enable automining in POA mode",
			Category:    CategoryBootstrap,
		},
		{
			Key:         "bootstrap-beacon-connection-timeout",
			Type:        TypeDuration,
			Default:     time.Minute,
			Description: "Timeout before emitting a warn log when connecting to bootstrapping beacons",
			Category:    CategoryBootstrap,
		},
		{
			Key:         "bootstrap-max-time-get-ancestors",
			Type:        TypeDuration,
			Default:     50 * time.Millisecond,
			Description: "Max Time to spend fetching a container and its ancestors when responding to a GetAncestors",
			Category:    CategoryBootstrap,
		},
		{
			Key:         "bootstrap-ancestors-max-containers-sent",
			Type:        TypeUint,
			Default:     uint(2000),
			Description: "Max number of containers in an Ancestors message sent by this node",
			Category:    CategoryBootstrap,
		},
		{
			Key:         "bootstrap-ancestors-max-containers-received",
			Type:        TypeUint,
			Default:     uint(2000),
			Description: "This node reads at most this many containers from an incoming Ancestors message",
			Category:    CategoryBootstrap,
		},

		// ============================================================================
		// Consensus Flags
		// ============================================================================
		{
			Key:         "consensus-sample-size",
			Type:        TypeInt,
			Default:     20,
			Description: "Number of nodes to query for each network poll",
			Category:    CategoryConsensus,
		},
		{
			Key:         "consensus-quorum-size",
			Type:        TypeInt,
			Default:     15,
			Description: "Threshold of nodes required to update this node's preference and increase its confidence in a network poll",
			Category:    CategoryConsensus,
		},
		{
			Key:         "consensus-preference-quorum-size",
			Type:        TypeInt,
			Default:     15,
			Description: "Threshold of nodes required to update this node's preference in a network poll. Ignored if consensus-quorum-size is provided",
			Category:    CategoryConsensus,
		},
		{
			Key:         "consensus-confidence-quorum-size",
			Type:        TypeInt,
			Default:     15,
			Description: "Threshold of nodes required to increase this node's confidence in a network poll. Ignored if consensus-quorum-size is provided",
			Category:    CategoryConsensus,
		},
		{
			Key:         "consensus-commit-threshold",
			Type:        TypeInt,
			Default:     20,
			Description: "Beta value to use for consensus",
			Category:    CategoryConsensus,
		},
		{
			Key:         "consensus-concurrent-repolls",
			Type:        TypeInt,
			Default:     4,
			Description: "Minimum number of concurrent polls for finalizing consensus",
			Category:    CategoryConsensus,
		},
		{
			Key:         "consensus-optimal-processing",
			Type:        TypeInt,
			Default:     50,
			Description: "Optimal number of processing containers in consensus",
			Category:    CategoryConsensus,
		},
		{
			Key:         "consensus-max-processing",
			Type:        TypeInt,
			Default:     1024,
			Description: "Maximum number of processing items to be considered healthy",
			Category:    CategoryConsensus,
		},
		{
			Key:         "consensus-max-time-processing",
			Type:        TypeDuration,
			Default:     30 * time.Second,
			Description: "Maximum amount of time an item should be processing and still be healthy",
			Category:    CategoryConsensus,
		},

		// ============================================================================
		// ProposerVM Flags
		// ============================================================================
		{
			Key:         "proposervm-use-current-height",
			Type:        TypeBool,
			Default:     false,
			Description: "Have the ProposerVM always report the last accepted P-chain block height",
			Category:    CategoryConsensus,
		},
		{
			Key:         "proposervm-min-block-delay",
			Type:        TypeDuration,
			Default:     time.Second,
			Description: "Minimum delay to enforce when building a chain++ block for the primary network chains and the default minimum delay for subnets",
			Category:    CategoryConsensus,
		},

		// ============================================================================
		// Metrics Flags
		// ============================================================================
		{
			Key:         "meter-vms-enabled",
			Type:        TypeBool,
			Default:     true,
			Description: "Enable Meter VMs to track VM performance with more granularity",
			Category:    CategoryMetrics,
		},
		{
			Key:         "uptime-metric-freq",
			Type:        TypeDuration,
			Default:     30 * time.Second,
			Description: "Frequency of renewing this node's average uptime metric",
			Category:    CategoryMetrics,
		},

		// ============================================================================
		// Index Flags
		// ============================================================================
		{
			Key:         "index-enabled",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, index all accepted containers and transactions and expose them via an API",
			Category:    CategoryIndex,
		},
		{
			Key:         "index-allow-incomplete",
			Type:        TypeBool,
			Default:     false,
			Description: "If true, allow running the node in such a way that could cause an index to miss transactions. Ignored if index is disabled",
			Category:    CategoryIndex,
		},

		// ============================================================================
		// Chain Config Flags
		// ============================================================================
		{
			Key:         "chain-config-dir",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/configs/chains",
			Description: "Chain specific configurations parent directory. Ignored if chain-config-content is specified",
			Category:    CategoryChain,
		},
		{
			Key:         "chain-config-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded chains configurations",
			Category:    CategoryChain,
		},
		{
			Key:         "subnet-config-dir",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/configs/subnets",
			Description: "Net specific configurations parent directory. Ignored if net-config-content is specified",
			Category:    CategoryChain,
		},
		{
			Key:         "net-config-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded subnets configurations",
			Category:    CategoryChain,
		},
		{
			Key:         "chain-data-dir",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/chainData",
			Description: "Chain specific data directory",
			Category:    CategoryChain,
		},
		{
			Key:         "import-chain-data",
			Type:        TypeString,
			Default:     "",
			Description: "Path to import blockchain data from another chain into C-Chain",
			Category:    CategoryChain,
		},

		// ============================================================================
		// Profile Flags
		// ============================================================================
		{
			Key:         "profile-dir",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/profiles",
			Description: "Path to the profile directory",
			Category:    CategoryProfile,
		},
		{
			Key:         "profile-continuous-enabled",
			Type:        TypeBool,
			Default:     false,
			Description: "Whether the app should continuously produce performance profiles",
			Category:    CategoryProfile,
		},
		{
			Key:         "profile-continuous-freq",
			Type:        TypeDuration,
			Default:     15 * time.Minute,
			Description: "How frequently to rotate performance profiles",
			Category:    CategoryProfile,
		},
		{
			Key:         "profile-continuous-max-files",
			Type:        TypeInt,
			Default:     5,
			Description: "Maximum number of historical profiles to keep",
			Category:    CategoryProfile,
		},

		// ============================================================================
		// Alias Flags
		// ============================================================================
		{
			Key:         "vm-aliases-file",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/configs/vms/aliases.json",
			Description: "Specifies a JSON file that maps vmIDs with custom aliases. Ignored if vm-aliases-file-content is specified",
			Category:    CategoryChain,
		},
		{
			Key:         "vm-aliases-file-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded maps vmIDs with custom aliases",
			Category:    CategoryChain,
		},
		{
			Key:         "chain-aliases-file",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/configs/chains/aliases.json",
			Description: "Specifies a JSON file that maps blockchainIDs with custom aliases. Ignored if chain-config-content is specified",
			Category:    CategoryChain,
		},
		{
			Key:         "chain-aliases-file-content",
			Type:        TypeString,
			Default:     "",
			Description: "Specifies base64 encoded map from blockchainID to custom aliases",
			Category:    CategoryChain,
		},

		// ============================================================================
		// Network Reconnect Flags
		// ============================================================================
		{
			Key:         "network-initial-reconnect-delay",
			Type:        TypeDuration,
			Default:     time.Second,
			Description: "Initial delay duration must be waited before attempting to reconnect a peer",
			Category:    CategoryNetwork,
		},
		{
			Key:         "network-max-reconnect-delay",
			Type:        TypeDuration,
			Default:     time.Hour,
			Description: "Maximum delay duration must be waited before attempting to reconnect a peer",
			Category:    CategoryNetwork,
		},

		// ============================================================================
		// System Tracker Flags
		// ============================================================================
		{
			Key:         "system-tracker-frequency",
			Type:        TypeDuration,
			Default:     500 * time.Millisecond,
			Description: "Frequency to check the real system usage of tracked processes. More frequent checks --> usage metrics are more accurate, but more expensive to track",
			Category:    CategorySystem,
		},
		{
			Key:         "system-tracker-processing-halflife",
			Type:        TypeDuration,
			Default:     15 * time.Second,
			Description: "Halflife to use for the processing requests tracker. Larger halflife --> usage metrics change more slowly",
			Category:    CategorySystem,
		},
		{
			Key:         "system-tracker-cpu-halflife",
			Type:        TypeDuration,
			Default:     15 * time.Second,
			Description: "Halflife to use for the cpu tracker. Larger halflife --> cpu usage metrics change more slowly",
			Category:    CategorySystem,
		},
		{
			Key:         "system-tracker-disk-halflife",
			Type:        TypeDuration,
			Default:     time.Minute,
			Description: "Halflife to use for the disk tracker. Larger halflife --> disk usage metrics change more slowly",
			Category:    CategorySystem,
		},
		{
			Key:         "system-tracker-disk-required-available-space",
			Type:        TypeUint64,
			Default:     uint64(512 * 1024 * 1024),
			Description: "Minimum number of available bytes on disk, under which the node will shutdown",
			Category:    CategorySystem,
		},
		{
			Key:         "system-tracker-disk-warning-threshold-available-space",
			Type:        TypeUint64,
			Default:     uint64(1024 * 1024 * 1024),
			Description: "Warning threshold for the number of available bytes on disk, under which the node will be considered unhealthy. Must be >= system-tracker-disk-required-available-space",
			Category:    CategorySystem,
		},

		// ============================================================================
		// CPU Management Flags
		// ============================================================================
		{
			Key:         "throttler-inbound-cpu-validator-alloc",
			Type:        TypeFloat64,
			Default:     float64(runtime.NumCPU()),
			Description: "Maximum number of CPUs to allocate for use by validators. Value should be in range [0, total core count]",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-inbound-cpu-max-non-validator-usage",
			Type:        TypeFloat64,
			Default:     0.8 * float64(runtime.NumCPU()),
			Description: "Number of CPUs that if fully utilized, will rate limit all non-validators. Value should be in range [0, total core count]",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-inbound-cpu-max-non-validator-node-usage",
			Type:        TypeFloat64,
			Default:     float64(runtime.NumCPU()) / 8,
			Description: "Maximum number of CPUs that a non-validator can utilize. Value should be in range [0, total core count]",
			Category:    CategoryThrottler,
		},

		// ============================================================================
		// Disk Management Flags
		// ============================================================================
		{
			Key:         "throttler-inbound-disk-validator-alloc",
			Type:        TypeFloat64,
			Default:     1000.0 * 1024 * 1024 * 1024,
			Description: "Maximum number of disk reads/writes per second to allocate for use by validators. Must be > 0",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-inbound-disk-max-non-validator-usage",
			Type:        TypeFloat64,
			Default:     1000.0 * 1024 * 1024 * 1024,
			Description: "Number of disk reads/writes per second that, if fully utilized, will rate limit all non-validators. Must be >= 0",
			Category:    CategoryThrottler,
		},
		{
			Key:         "throttler-inbound-disk-max-non-validator-node-usage",
			Type:        TypeFloat64,
			Default:     1000.0 * 1024 * 1024 * 1024,
			Description: "Maximum number of disk reads/writes per second that a non-validator can utilize. Must be >= 0",
			Category:    CategoryThrottler,
		},

		// ============================================================================
		// Tracing Flags
		// ============================================================================
		{
			Key:         "tracing-exporter-type",
			Type:        TypeString,
			Default:     "disabled",
			Description: "Type of exporter to use for tracing. Options are [disabled, grpc, http]",
			Category:    CategoryTracing,
			Constraints: &Constraints{
				Enum: []string{"disabled", "grpc", "http"},
			},
		},
		{
			Key:         "tracing-endpoint",
			Type:        TypeString,
			Default:     "",
			Description: "The endpoint to send trace data to. If unspecified, the default endpoint will be used; depending on the exporter type",
			Category:    CategoryTracing,
		},
		{
			Key:         "tracing-insecure",
			Type:        TypeBool,
			Default:     true,
			Description: "If true, don't use TLS when sending trace data",
			Category:    CategoryTracing,
		},
		{
			Key:         "tracing-sample-rate",
			Type:        TypeFloat64,
			Default:     0.1,
			Description: "The fraction of traces to sample. If >= 1, always sample. If <= 0, never sample",
			Category:    CategoryTracing,
		},
		{
			Key:         "tracing-headers",
			Type:        TypeStringToString,
			Default:     map[string]string{},
			Description: "The headers to provide the trace indexer",
			Category:    CategoryTracing,
		},
		{
			Key:         "process-context-file",
			Type:        TypeString,
			Default:     "$LUXD_DATA_DIR/process.json",
			Description: "The path to write process context to (including PID, API URI, and staking address)",
			Category:    CategoryNode,
		},

		// ============================================================================
		// POA Mode Flags
		// ============================================================================
		{
			Key:         "poa-mode-enabled",
			Type:        TypeBool,
			Default:     false,
			Description: "Enable Proof of Authority mode for subnets",
			Category:    CategoryPOA,
		},
		{
			Key:         "poa-single-node-mode",
			Type:        TypeBool,
			Default:     false,
			Description: "Enable single node POA mode (no consensus required)",
			Category:    CategoryPOA,
		},
		{
			Key:         "poa-min-block-time",
			Type:        TypeDuration,
			Default:     time.Second,
			Description: "Minimum time between blocks in POA mode",
			Category:    CategoryPOA,
		},
		{
			Key:         "poa-authorized-nodes",
			Type:        TypeStringSlice,
			Default:     nil,
			Description: "List of authorized nodes for POA mode",
			Category:    CategoryPOA,
		},
		{
			Key:         "force-ignore-checksum",
			Type:        TypeBool,
			Default:     false,
			Description: "Force ignore checksum validation errors (use with caution)",
			Category:    CategoryDev,
		},
	}
}
