// Copyright (C) 2022, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

// Package fs provides filesystem utility functions.
package fs

import (
	"os"
	"path/filepath"
	"strings"
)

// FileExists returns true if the file exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists returns true if the directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsSubPath checks if child is a subdirectory of parent.
// Returns true if parent is a prefix path of child.
func IsSubPath(child, parent string) bool {
	// Clean both paths
	cleanChild, err := filepath.Abs(child)
	if err != nil {
		return false
	}
	cleanParent, err := filepath.Abs(parent)
	if err != nil {
		return false
	}

	// Ensure parent has trailing separator for proper prefix matching
	if !strings.HasSuffix(cleanParent, string(filepath.Separator)) {
		cleanParent += string(filepath.Separator)
	}

	return strings.HasPrefix(cleanChild, cleanParent)
}
