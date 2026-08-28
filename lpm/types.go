// Copyright (C) 2025, Lux Industries Inc All rights reserved.

package lpm

type Metadata struct {
	Alias       string
	Homepage    string
	Description string
	Maintainers []string
}

type VMUpload struct {
	ID              string
	Alias           string
	Homepage        string
	Description     string
	BinaryPath      string
	InstallScript   string
	ChainConfigPath string
	GenesisPath     string
	ReadmePath      string
	LicensePath     string
	ChainPath       string
	Versions        []string
}

type Chain struct {
	ID          string
	Alias       string
	VM          string
	Config      string
	Genesis     string
	Description string
}

type VM struct {
	ID          string
	Alias       string
	VMType      string
	Binary      string
	ChainConfig string
	Chain       string
	Genesis     string
	Version     string
	URL         string
	Checksum    string
	Runtime     string
	Description string
}
