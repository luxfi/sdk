// Copyright (C) 2019-2022, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"path/filepath"
	"strings"
)

// GetFilesAndAliases gets files matching a glob pattern and returns them with aliases
func GetFilesAndAliases(pattern string) (map[string]string, error) {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	
	aliases := make(map[string]string)
	for _, file := range files {
		base := filepath.Base(file)
		alias := strings.TrimSuffix(base, filepath.Ext(base))
		aliases[alias] = file
	}
	
	return aliases, nil
}

// SubnetsSuite manages subnet lifecycle for tests
type SubnetsSuite struct {
	blockchainIDs map[string]string
}

// CreateSubnetsSuite creates a new subnet suite from genesis files
func CreateSubnetsSuite(genesisFiles map[string]string) *SubnetsSuite {
	// In a real implementation, this would create subnets from genesis files
	// For now, return a mock implementation
	return &SubnetsSuite{
		blockchainIDs: make(map[string]string),
	}
}

// GetBlockchainID returns the blockchain ID for a given alias
func (s *SubnetsSuite) GetBlockchainID(alias string) string {
	// In a real implementation, this would return the actual blockchain ID
	// For now, return a mock ID
	return "2f1YmBU5jmfuXbjRYKEch2kjQwsKdHCPJfMCoKq8CUNYrLCPTa"
}

