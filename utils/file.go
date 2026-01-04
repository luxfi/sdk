// Copyright (C) 2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"os"
	"path/filepath"
)

// FileExists checks if a file exists.
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists.
func DirExists(dirName string) bool {
	info, err := os.Stat(dirName)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// ExpandHome expands ~ symbol to home directory
func ExpandHome(path string) string {
	if path == "" {
		home, _ := os.UserHomeDir()
		return home
	}
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}
	return path
}

// IsSubPath checks if childPath is inside parentPath.
// Both paths should be absolute paths.
// Returns true if childPath is a subdirectory or file inside parentPath.
func IsSubPath(childPath, parentPath string) bool {
	// Clean and normalize paths
	child := filepath.Clean(childPath)
	parent := filepath.Clean(parentPath)

	// Get relative path from parent to child
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	// If the relative path starts with "..", child is not inside parent
	// Also, if rel equals ".", they're the same path
	return rel != "." && len(rel) > 0 && rel[0] != '.'
}
