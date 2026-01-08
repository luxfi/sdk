// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package packages

import (
	"fmt"

	"golang.org/x/tools/go/packages"

	"github.com/luxfi/math/set"
)

// GetDependencies takes a fully qualified package name and returns a map of all
// its recursive package imports (including itself) in the same format.
func GetDependencies(packageName string) (set.Set[string], error) {
	// Configure the load mode to include dependencies
	cfg := &packages.Config{Mode: packages.NeedImports | packages.NeedName}
	pkgs, err := packages.Load(cfg, packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for %s", packageName)
	}

	// Initialize deps set
	deps := set.NewSet[string](1)

	var collectDeps func(pkg *packages.Package) // collectDeps is recursive
	collectDeps = func(pkg *packages.Package) {
		if deps.Contains(pkg.PkgPath) {
			return // Avoid re-processing the same dependency
		}
		deps.Add(pkg.PkgPath)
		for _, dep := range pkg.Imports {
			collectDeps(dep)
		}
	}

	// Start collecting dependencies
	for _, pkg := range pkgs {
		if pkg.Errors != nil {
			return nil, fmt.Errorf("failed to load package %s, %v", packageName, pkg.Errors)
		}
		collectDeps(pkg)
	}
	return deps, nil
}

// GetDirectImports takes a fully qualified package name and returns a set of
// its direct package imports (non-recursive) in the same format.
func GetDirectImports(packageName string) (set.Set[string], error) {
	// Configure the load mode to include imports
	cfg := &packages.Config{Mode: packages.NeedImports | packages.NeedName}
	pkgs, err := packages.Load(cfg, packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for %s", packageName)
	}

	// Initialize imports set
	imports := set.NewSet[string](1)

	// Collect direct imports from all matching packages
	for _, pkg := range pkgs {
		if pkg.Errors != nil {
			return nil, fmt.Errorf("failed to load package %s, %v", packageName, pkg.Errors)
		}
		// Add direct imports only
		for importPath := range pkg.Imports {
			imports.Add(importPath)
		}
	}
	return imports, nil
}
