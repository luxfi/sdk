// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package admin

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/gorilla/rpc/v2"

	"github.com/luxfi/const"
	"github.com/luxfi/database"
	"github.com/luxfi/ids"
	"github.com/luxfi/log"
	"github.com/luxfi/sdk/node/api"
	"github.com/luxfi/sdk/node/api/server"
	"github.com/luxfi/sdk/node/chains"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/sdk/utils/formatting"
	"github.com/luxfi/sdk/utils/json"
	"github.com/luxfi/sdk/utils/perms"
	"github.com/luxfi/sdk/utils/profiler"
	"github.com/luxfi/sdk/node/vms"
	"github.com/luxfi/sdk/node/vms/registry"
	"github.com/luxfi/math/set"
)

const (
	maxAliasLength = 512

	// Name of file that stacktraces are written to
	stacktraceFile = "stacktrace.txt"
)

var (
	errAliasTooLong = errors.New("alias length is too long")
	errNoLogLevel   = errors.New("need to specify either displayLevel or logLevel")
)

// ChainTracker is the interface for tracking chains at runtime
type ChainTracker interface {
	TrackChain(chainID ids.ID) error
	TrackedChains() set.Set[ids.ID]
}

type Config struct {
	Log          log.Logger
	ProfileDir   string
	LogFactory   log.Factory
	NodeConfig   interface{}
	DB           database.Database
	ChainManager chains.Manager
	HTTPServer   server.PathAdderWithReadLock
	VMRegistry   registry.VMRegistry
	VMManager    vms.Manager
	PluginDir    string
	Network      ChainTracker
}

// Admin is the API service for node admin management
type Admin struct {
	Config
	lock     sync.RWMutex
	profiler profiler.Profiler
}

// NewService returns a new admin API service.
// All of the fields in [config] must be set.
func NewService(config Config) (http.Handler, error) {
	server := rpc.NewServer()
	codec := json.NewCodec()
	server.RegisterCodec(codec, "application/json")
	server.RegisterCodec(codec, "application/json;charset=UTF-8")
	return server, server.RegisterService(
		&Admin{
			Config:   config,
			profiler: profiler.New(config.ProfileDir),
		},
		"admin",
	)
}

// StartCPUProfiler starts a cpu profile writing to the specified file
func (a *Admin) StartCPUProfiler(_ *http.Request, _ *struct{}, _ *api.EmptyReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "startCPUProfiler"),
	)

	a.lock.Lock()
	defer a.lock.Unlock()

	return a.profiler.StartCPUProfiler()
}

// StopCPUProfiler stops the cpu profile
func (a *Admin) StopCPUProfiler(_ *http.Request, _ *struct{}, _ *api.EmptyReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "stopCPUProfiler"),
	)

	a.lock.Lock()
	defer a.lock.Unlock()

	return a.profiler.StopCPUProfiler()
}

// MemoryProfile runs a memory profile writing to the specified file
func (a *Admin) MemoryProfile(_ *http.Request, _ *struct{}, _ *api.EmptyReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "memoryProfile"),
	)

	a.lock.Lock()
	defer a.lock.Unlock()

	return a.profiler.MemoryProfile()
}

// LockProfile runs a mutex profile writing to the specified file
func (a *Admin) LockProfile(_ *http.Request, _ *struct{}, _ *api.EmptyReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "lockProfile"),
	)

	a.lock.Lock()
	defer a.lock.Unlock()

	return a.profiler.LockProfile()
}

// Alias attempts to alias an HTTP endpoint to a new name
func (a *Admin) Alias(_ *http.Request, args *AliasArgs, _ *api.EmptyReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "alias"),
		log.String("endpoint", args.Endpoint),
		log.String("alias", args.Alias),
	)

	if len(args.Alias) > maxAliasLength {
		return errAliasTooLong
	}

	return a.HTTPServer.AddAliasesWithReadLock(args.Endpoint, args.Alias)
}

// AliasChain attempts to alias a chain to a new name
func (a *Admin) AliasChain(_ *http.Request, args *AliasChainArgs, _ *api.EmptyReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "aliasChain"),
		log.String("chain", args.Chain),
		log.String("alias", args.Alias),
	)

	if len(args.Alias) > maxAliasLength {
		return errAliasTooLong
	}
	chainID, err := a.ChainManager.Lookup(args.Chain)
	if err != nil {
		return err
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	if err := a.ChainManager.Alias(chainID, args.Alias); err != nil {
		return err
	}

	endpoint := path.Join(constants.ChainAliasPrefix, chainID.String())
	alias := path.Join(constants.ChainAliasPrefix, args.Alias)
	return a.HTTPServer.AddAliasesWithReadLock(endpoint, alias)
}

// GetChainAliases returns the aliases of the chain
func (a *Admin) GetChainAliases(_ *http.Request, args *GetChainAliasesArgs, reply *GetChainAliasesReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "getChainAliases"),
		log.String("chain", args.Chain),
	)

	id, err := ids.FromString(args.Chain)
	if err != nil {
		return err
	}

	reply.Aliases, err = a.ChainManager.Aliases(id)
	return err
}

// Stacktrace returns the current global stacktrace
func (a *Admin) Stacktrace(_ *http.Request, _ *struct{}, _ *api.EmptyReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "stacktrace"),
	)

	stacktrace := []byte(utils.GetStacktrace(true))

	a.lock.Lock()
	defer a.lock.Unlock()

	return perms.WriteFile(stacktraceFile, stacktrace, perms.ReadWrite)
}

// SetLoggerLevel sets the log level and/or display level for loggers.
// If len([args.LoggerName]) == 0, sets the log/display level of all loggers.
// Otherwise, sets the log/display level of the loggers named in that argument.
// Sets the log level of these loggers to args.LogLevel.
// If args.LogLevel == nil, doesn't set the log level of these loggers.
// If args.LogLevel != nil, must be a valid string representation of a log level.
// Sets the display level of these loggers to args.LogLevel.
// If args.DisplayLevel == nil, doesn't set the display level of these loggers.
// If args.DisplayLevel != nil, must be a valid string representation of a log level.
func (a *Admin) SetLoggerLevel(_ *http.Request, args *SetLoggerLevelArgs, reply *LoggerLevelReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "setLoggerLevel"),
		log.String("loggerName", args.LoggerName),
		log.Stringer("logLevel", args.LogLevel),
		log.Stringer("displayLevel", args.DisplayLevel),
	)

	if args.LogLevel == nil && args.DisplayLevel == nil {
		return errNoLogLevel
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	loggerNames := a.getLoggerNames(args.LoggerName)
	// LogFactory methods not available in new log module
	// for _, name := range loggerNames {
	// 	if args.LogLevel != nil {
	// 		if err := a.LogFactory.SetLogLevel(name, *args.LogLevel); err != nil {
	// 			return err
	// 		}
	// 	}
	// 	if args.DisplayLevel != nil {
	// 		if err := a.LogFactory.SetDisplayLevel(name, *args.DisplayLevel); err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	var err error
	reply.LoggerLevels, err = a.getLogLevels(loggerNames)
	return err
}

// GetLoggerLevel returns the log level and display level of all loggers.
func (a *Admin) GetLoggerLevel(_ *http.Request, args *GetLoggerLevelArgs, reply *LoggerLevelReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "getLoggerLevel"),
		log.String("loggerName", args.LoggerName),
	)

	a.lock.RLock()
	defer a.lock.RUnlock()

	loggerNames := a.getLoggerNames(args.LoggerName)

	var err error
	reply.LoggerLevels, err = a.getLogLevels(loggerNames)
	return err
}

// GetConfig returns the config that the node was started with.
func (a *Admin) GetConfig(_ *http.Request, _ *struct{}, reply *interface{}) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "getConfig"),
	)
	*reply = a.NodeConfig
	return nil
}

// LoadVMs loads any new VMs available to the node and returns the added VMs.
// After loading new VMs, it retries creating any chains that were waiting for
// those VMs (hot-loading support).
func (a *Admin) LoadVMs(r *http.Request, _ *struct{}, reply *LoadVMsReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "loadVMs"),
	)

	a.lock.Lock()
	defer a.lock.Unlock()

	ctx := r.Context()
	loadedVMs, failedVMs, err := a.VMRegistry.Reload(ctx)
	if err != nil {
		return err
	}

	// extract the inner error messages
	failedVMsParsed := make(map[ids.ID]string)
	for vmID, err := range failedVMs {
		failedVMsParsed[vmID] = err.Error()
	}

	reply.FailedVMs = failedVMsParsed
	reply.NewVMs, err = ids.GetRelevantAliases(a.VMManager, loadedVMs)
	if err != nil {
		return err
	}

	// Hot-loading: retry chains that were waiting for these VMs
	totalRetried := 0
	for _, vmID := range loadedVMs {
		retried := a.ChainManager.RetryPendingChains(vmID)
		if retried > 0 {
			a.Log.Info("Retrying pending chains after VM hot-load",
				log.Stringer("vmID", vmID),
				log.Int("chainsRetried", retried),
			)
		}
		totalRetried += retried
	}
	reply.ChainsRetried = totalRetried

	return nil
}

func (a *Admin) getLoggerNames(loggerName string) []string {
	if len(loggerName) == 0 {
		// LogFactory.GetLoggerNames not available
		return []string{}
	}
	return []string{loggerName}
}

func (a *Admin) getLogLevels(loggerNames []string) (map[string]LogAndDisplayLevels, error) {
	loggerLevels := make(map[string]LogAndDisplayLevels)
	// LogFactory methods not available
	// for _, name := range loggerNames {
	// 	logLevel, err := a.LogFactory.GetLogLevel(name)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	displayLevel, err := a.LogFactory.GetDisplayLevel(name)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	loggerLevels[name] = LogAndDisplayLevels{
	// 		LogLevel:     logLevel,
	// 		DisplayLevel: displayLevel,
	// 	}
	// }
	return loggerLevels, nil
}

//nolint:staticcheck // renaming this method to DBGet would change the API method from "dbGet" to "dBGet"
func (a *Admin) DbGet(_ *http.Request, args *DBGetArgs, reply *DBGetReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "dbGet"),
		log.String("key", args.Key),
	)

	key, err := formatting.Decode(formatting.HexNC, args.Key)
	if err != nil {
		return err
	}

	value, err := a.DB.Get(key)
	if err != nil {
		return err
	}

	reply.Value, err = formatting.Encode(formatting.HexNC, value)
	return err
}

// ListVMs returns all registered VMs with their IDs, aliases, and paths
func (a *Admin) ListVMs(_ *http.Request, _ *struct{}, reply *ListVMsReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "listVMs"),
	)

	a.lock.RLock()
	defer a.lock.RUnlock()

	// Get all registered VM IDs
	vmIDs, err := a.VMManager.ListFactories()
	if err != nil {
		return err
	}

	// Build a map of plugin files in the plugin directory for path lookup
	pluginPaths := make(map[ids.ID]string)
	if a.PluginDir != "" {
		entries, err := os.ReadDir(a.PluginDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				// Strip extension for matching
				baseName := name[:len(name)-len(filepath.Ext(name))]
				if baseName == "" {
					continue
				}
				// Try to parse as VM ID
				vmID, err := ids.FromString(baseName)
				if err == nil {
					pluginPaths[vmID] = filepath.Join(a.PluginDir, name)
				}
			}
		}
	}

	reply.VMs = make(map[string]VMInfo, len(vmIDs))
	for _, vmID := range vmIDs {
		aliases, err := a.VMManager.Aliases(vmID)
		if err != nil {
			return err
		}

		// Filter out the vmID string from aliases (it's always included)
		vmIDStr := vmID.String()
		filteredAliases := make([]string, 0, len(aliases))
		for _, alias := range aliases {
			if alias != vmIDStr {
				filteredAliases = append(filteredAliases, alias)
			}
		}

		info := VMInfo{
			ID:      vmIDStr,
			Aliases: filteredAliases,
		}

		// Add path if found in plugin directory
		if pluginPath, ok := pluginPaths[vmID]; ok {
			info.Path = pluginPath
		}

		reply.VMs[vmIDStr] = info
	}

	return nil
}

// SetTrackedChains adds chains to be tracked by this node at runtime.
// This enables the node to track new chains without requiring a restart.
func (a *Admin) SetTrackedChains(_ *http.Request, args *SetTrackedChainsArgs, reply *SetTrackedChainsReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "setTrackedChains"),
	)

	if a.Network == nil {
		return errors.New("network not available")
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	for _, chainStr := range args.Chains {
		chainID, err := ids.FromString(chainStr)
		if err != nil {
			return fmt.Errorf("invalid chain ID %q: %w", chainStr, err)
		}

		if err := a.Network.TrackChain(chainID); err != nil {
			return fmt.Errorf("failed to track chain %s: %w", chainID, err)
		}

		a.Log.Info("chain now tracked",
			log.Stringer("chainID", chainID),
		)
	}

	// Return the updated list of tracked chains
	trackedChains := a.Network.TrackedChains()
	reply.TrackedChains = make([]string, 0, trackedChains.Len())
	for chainID := range trackedChains {
		reply.TrackedChains = append(reply.TrackedChains, chainID.String())
	}

	return nil
}

// GetTrackedChains returns the list of chains currently being tracked by this node.
func (a *Admin) GetTrackedChains(_ *http.Request, _ *struct{}, reply *GetTrackedChainsReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "getTrackedChains"),
	)

	if a.Network == nil {
		return errors.New("network not available")
	}

	a.lock.RLock()
	defer a.lock.RUnlock()

	trackedChains := a.Network.TrackedChains()
	reply.TrackedChains = make([]string, 0, trackedChains.Len())
	for chainID := range trackedChains {
		reply.TrackedChains = append(reply.TrackedChains, chainID.String())
	}

	return nil
}

// Snapshot creates a database snapshot at the given path
func (a *Admin) Snapshot(_ *http.Request, args *SnapshotArgs, reply *SnapshotReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "snapshot"),
		log.String("path", args.Path),
		log.Uint64("since", args.Since),
	)

	if len(args.Path) == 0 {
		return errors.New("snapshot path required")
	}

	// Ensure directory exists
	dir := filepath.Dir(args.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	f, err := os.Create(args.Path)
	if err != nil {
		return fmt.Errorf("failed to create snapshot file: %w", err)
	}
	defer f.Close()

	version, err := a.DB.Backup(f, args.Since)
	if err != nil {
		return fmt.Errorf("snapshot failed: %w", err)
	}

	reply.Success = true
	reply.Version = version
	return nil
}

// Load restores the database from the given snapshot path
func (a *Admin) Load(_ *http.Request, args *LoadArgs, _ *api.EmptyReply) error {
	a.Log.Debug("API called",
		log.String("service", "admin"),
		log.String("method", "load"),
		log.String("path", args.Path),
	)

	if len(args.Path) == 0 {
		return errors.New("snapshot path required")
	}

	f, err := os.Open(args.Path)
	if err != nil {
		return fmt.Errorf("failed to open snapshot file: %w", err)
	}
	defer f.Close()

	if err := a.DB.Load(f); err != nil {
		return fmt.Errorf("load failed: %w", err)
	}

	return nil
}
