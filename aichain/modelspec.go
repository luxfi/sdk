// Copyright (C) 2026, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package aichain

import (
	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
)

// ModelSpec is the SDK's canonical description of a model the chain can serve.
// On-chain (chains/aivm, modelregistry) a model is referred to ONLY by its
// 32-byte weight-commitment hash (an opaque equality key); ModelSpec is the
// human-facing identity that hashes DOWN to that key, so a requester naming a
// model and a provider advertising one derive the IDENTICAL modelSpecHash from
// the IDENTICAL fields.
//
// Fields:
//   - Name:           model identity, e.g. "zenlm/zen-omni" (free-form, UTF-8).
//   - Version:        monotonically increasing version the registry adopts.
//   - WeightCommit:   the 32-byte commitment to the exact weights (a content
//     hash of the weight file / a Merkle root over shards). This is the
//     load-bearing field — it pins WHICH weights are canonical. If you already
//     have the on-chain commitment, set it directly; Name/Version then only
//     decorate it.
//   - Quantization:   the inference format that makes the result deterministic,
//     e.g. "int8", "bf16-kat". Different quantizations of the same weights are
//     different specs (they produce different canonical outputs).
type ModelSpec struct {
	Name         string      `json:"name"`
	Version      uint64      `json:"version"`
	WeightCommit common.Hash `json:"weightCommit"`
	Quantization string      `json:"quantization"`
}

// Hash is the canonical model-spec hash — the 32-byte value carried as
// modelSpecHash in an intent and adopted (as weightHash) in the registry:
//
//	keccak256( DomainModelSpec ||
//	    u32be(len(Name)) || Name ||
//	    u64be(Version) ||
//	    WeightCommit(32) ||
//	    u32be(len(Quantization)) || Quantization )
//
// Variable-length fields are length-prefixed so no two distinct specs can
// concatenate to the same preimage (injective). The domain separator keeps this
// keyspace disjoint from intent ids and receipt hashes.
func (m ModelSpec) Hash() common.Hash {
	buf := make([]byte, 0, len(DomainModelSpec)+4+len(m.Name)+8+32+4+len(m.Quantization))
	buf = append(buf, []byte(DomainModelSpec)...)
	buf = append(buf, u32be(uint32(len(m.Name)))...)
	buf = append(buf, []byte(m.Name)...)
	buf = append(buf, u64be(m.Version)...)
	buf = append(buf, m.WeightCommit.Bytes()...)
	buf = append(buf, u32be(uint32(len(m.Quantization)))...)
	buf = append(buf, []byte(m.Quantization)...)
	return common.BytesToHash(crypto.Keccak256(buf))
}

// RegistryName is the bytes32 model name the registry's adopt(name, version,
// weightHash) ABI keys on. We use the spec Hash itself as the registry name so a
// model is addressed by one stable id everywhere (the registry maps that id ->
// the adopted weight commitment + version).
func (m ModelSpec) RegistryName() common.Hash { return m.Hash() }
