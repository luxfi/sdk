// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package profiler provides CPU, memory, and lock profiling utilities.
package profiler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/luxfi/vm/utils/perms"
)

const (
	cpuProfileFile  = "cpu.profile"
	memProfileFile  = "mem.profile"
	lockProfileFile = "lock.profile"
)

var (
	_ Profiler = (*profiler)(nil)

	errCPUProfilerRunning    = errors.New("cpu profiler already running")
	errCPUProfilerNotRunning = errors.New("cpu profiler doesn't exist")
)

// Profiler provides methods for measuring process performance.
type Profiler interface {
	StartCPUProfiler() error
	StopCPUProfiler() error
	MemoryProfile() error
	LockProfile() error
}

type profiler struct {
	dir             string
	cpuProfileName  string
	memProfileName  string
	lockProfileName string
	cpuProfileFile  *os.File
}

// New returns a new Profiler that writes to the given directory.
func New(dir string) Profiler {
	return newProfiler(dir)
}

func newProfiler(dir string) *profiler {
	return &profiler{
		dir:             dir,
		cpuProfileName:  filepath.Join(dir, cpuProfileFile),
		memProfileName:  filepath.Join(dir, memProfileFile),
		lockProfileName: filepath.Join(dir, lockProfileFile),
	}
}

func (p *profiler) StartCPUProfiler() error {
	if p.cpuProfileFile != nil {
		return errCPUProfilerRunning
	}

	if err := os.MkdirAll(p.dir, perms.ReadWriteExecute); err != nil {
		return err
	}
	file, err := perms.Create(p.cpuProfileName, perms.ReadWrite)
	if err != nil {
		return err
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		file.Close()
		return err
	}
	p.cpuProfileFile = file
	return nil
}

func (p *profiler) StopCPUProfiler() error {
	if p.cpuProfileFile == nil {
		return errCPUProfilerNotRunning
	}

	pprof.StopCPUProfile()
	err := p.cpuProfileFile.Close()
	p.cpuProfileFile = nil
	return err
}

func (p *profiler) MemoryProfile() error {
	if err := os.MkdirAll(p.dir, perms.ReadWriteExecute); err != nil {
		return err
	}

	file, err := perms.Create(p.memProfileName, perms.ReadWrite)
	if err != nil {
		return err
	}
	defer file.Close()

	runtime.GC()
	return pprof.WriteHeapProfile(file)
}

func (p *profiler) LockProfile() error {
	if err := os.MkdirAll(p.dir, perms.ReadWriteExecute); err != nil {
		return err
	}

	file, err := perms.Create(p.lockProfileName, perms.ReadWrite)
	if err != nil {
		return err
	}
	defer file.Close()

	profile := pprof.Lookup("mutex")
	if profile == nil {
		return errors.New("mutex profile not found")
	}
	return profile.WriteTo(file, 0)
}

// Config for continuous profiler.
type Config struct {
	Dir         string        `json:"dir"`
	Enabled     bool          `json:"enabled"`
	Freq        time.Duration `json:"freq"`
	MaxNumFiles int           `json:"maxNumFiles"`
}

// ContinuousProfiler periodically captures profiles.
type ContinuousProfiler interface {
	Dispatch() error
	Shutdown()
}

type continuousProfiler struct {
	profiler    *profiler
	freq        time.Duration
	maxNumFiles int
	closer      chan struct{}
}

// NewContinuous returns a new continuous profiler.
func NewContinuous(dir string, freq time.Duration, maxNumFiles int) ContinuousProfiler {
	return &continuousProfiler{
		profiler:    newProfiler(dir),
		freq:        freq,
		maxNumFiles: maxNumFiles,
		closer:      make(chan struct{}),
	}
}

func (p *continuousProfiler) Dispatch() error {
	t := time.NewTicker(p.freq)
	defer t.Stop()

	for {
		if err := p.start(); err != nil {
			return err
		}

		select {
		case <-p.closer:
			return p.stop()
		case <-t.C:
			if err := p.stop(); err != nil {
				return err
			}
		}

		if err := p.rotate(); err != nil {
			return err
		}
	}
}

func (p *continuousProfiler) start() error {
	return p.profiler.StartCPUProfiler()
}

func (p *continuousProfiler) stop() error {
	g := errgroup.Group{}
	g.Go(p.profiler.StopCPUProfiler)
	g.Go(p.profiler.MemoryProfile)
	g.Go(p.profiler.LockProfile)
	return g.Wait()
}

func (p *continuousProfiler) rotate() error {
	g := errgroup.Group{}
	g.Go(func() error { return rotate(p.profiler.cpuProfileName, p.maxNumFiles) })
	g.Go(func() error { return rotate(p.profiler.memProfileName, p.maxNumFiles) })
	g.Go(func() error { return rotate(p.profiler.lockProfileName, p.maxNumFiles) })
	return g.Wait()
}

func (p *continuousProfiler) Shutdown() {
	close(p.closer)
}

func rotate(name string, maxNumFiles int) error {
	for i := maxNumFiles - 1; i > 0; i-- {
		src := fmt.Sprintf("%s.%d", name, i)
		dst := fmt.Sprintf("%s.%d", name, i+1)
		if err := renameIfExists(src, dst); err != nil {
			return err
		}
	}
	return renameIfExists(name, name+".1")
}

func renameIfExists(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return os.Rename(src, dst)
}
