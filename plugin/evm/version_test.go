// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/node/version"
	"github.com/stretchr/testify/require"
)

type rpcChainCompatibility struct {
	RPCChainVMProtocolVersion map[string]uint `json:"rpcChainVMProtocolVersion"`
}

const compatibilityFile = "../../compatibility.json"

func TestCompatibility(t *testing.T) {
	compat, err := os.ReadFile(compatibilityFile)
	require.NoError(t, err, "reading compatibility file")

	var parsedCompat rpcChainCompatibility
	err = json.Unmarshal(compat, &parsedCompat)
	require.NoError(t, err, "json decoding compatibility file")

	rpcChainVMVersion, valueInJSON := parsedCompat.RPCChainVMProtocolVersion[Version]
	if !valueInJSON {
		t.Fatalf("%s has evm version %s missing from rpcChainVMProtocolVersion object",
			filepath.Base(compatibilityFile), Version)
	}
	if rpcChainVMVersion != version.RPCChainVMProtocol {
		t.Fatalf("%s has evm version %s stated as compatible with RPC chain VM protocol version %d but Luxd protocol version is %d",
			filepath.Base(compatibilityFile), Version, rpcChainVMVersion, version.RPCChainVMProtocol)
	}
}
