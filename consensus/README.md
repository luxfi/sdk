# Quasar Consensus SDK

The Quasar consensus module provides quantum-secure consensus primitives for building blockchain applications on the Lux network. This SDK encapsulates the Quasar protocol - a 2-round BFT consensus achieving sub-second finality with both classical (BLS) and post-quantum (ML-DSA/Corona) security.

## Overview

Quasar unifies all chain types (DAG, linear, EVM) under a single quantum-resistant consensus engine. The protocol operates through five physics-inspired phases:

1. **Photon**: VRF-based committee selection with luminance weighting
2. **Wave**: 2-round voting (BLS aggregation + lattice signatures)
3. **Focus**: Confidence convergence through beta-threshold agreement
4. **Prism**: DAG structure for parallel transaction processing
5. **Horizon**: Quantum finality anchoring with immutable certificates

## Installation

```go
import (
    "github.com/luxfi/sdk/consensus"
    "github.com/luxfi/sdk/consensus/signature"
)
```

## Core Interfaces

### Signature Interface

The `Signature` interface provides a unified API for all cryptographic signature schemes used in Quasar consensus:

```go
// SignatureType represents the cryptographic scheme used
type SignatureType uint8

const (
    // Classical signatures
    SignatureTypeBLS      SignatureType = iota // BLS12-381 aggregate signatures
    SignatureTypeSECP256K1                      // secp256k1 (Ethereum compatible)

    // Post-quantum signatures (NIST standards)
    SignatureTypeMLDSA87  // ML-DSA-87 (Dilithium5 equivalent) - NIST Level 5
    SignatureTypeMLDSA65  // ML-DSA-65 (Dilithium3 equivalent) - NIST Level 3
    SignatureTypeMLDSA44  // ML-DSA-44 (Dilithium2 equivalent) - NIST Level 2

    // Hybrid signatures (classical + PQC)
    SignatureTypeHybrid   // BLS + ML-DSA combined
    SignatureTypeCorona // Lattice-based threshold signatures
)

// Signature is the core interface for all signature operations
type Signature interface {
    // Type returns the signature scheme type
    Type() SignatureType

    // Sign signs a message and returns the signature bytes
    Sign(msg []byte) ([]byte, error)

    // Verify verifies a signature against a message
    Verify(msg, sig []byte) bool

    // PublicKey returns the public key bytes
    PublicKey() []byte

    // Bytes returns the raw signature bytes
    Bytes() []byte
}
```

### ThresholdSigner Interface

The `ThresholdSigner` interface supports distributed key generation and threshold signing for consensus:

```go
// ThresholdSigner provides threshold signature operations
type ThresholdSigner interface {
    Signature

    // Threshold returns the minimum signers required
    Threshold() int

    // TotalSigners returns the total number of signers
    TotalSigners() int

    // CreateShare generates a signature share for distributed signing
    CreateShare(msg []byte) (*SignatureShare, error)

    // CombineShares combines threshold shares into a full signature
    CombineShares(shares []*SignatureShare) ([]byte, error)

    // VerifyShare verifies an individual signature share
    VerifyShare(msg []byte, share *SignatureShare) bool

    // Precompute generates precomputation data for fast signing
    Precompute() ([]byte, error)

    // QuickSign signs using precomputed data (sub-millisecond)
    QuickSign(precomp, msg []byte) ([]byte, error)
}

// SignatureShare represents a partial signature from one signer
type SignatureShare struct {
    Index     uint32 // Signer index (1-based)
    Share     []byte // Partial signature
    PublicKey []byte // Signer's public key
}
```

## Signature Types

### BLS Signatures

BLS signatures provide efficient aggregation for classical consensus:

```go
import "github.com/luxfi/sdk/consensus/signature"

// Create a new BLS signer
signer, err := signature.NewBLS()
if err != nil {
    return err
}

// Sign a message
msg := []byte("block hash")
sig, err := signer.Sign(msg)
if err != nil {
    return err
}

// Verify signature
valid := signer.Verify(msg, sig)
```

**BLS Aggregation**:

```go
// Aggregate multiple BLS signatures (O(1) size)
signatures := [][]byte{sig1, sig2, sig3}
publicKeys := [][]byte{pk1, pk2, pk3}

aggregated, err := signature.AggregateBLS(signatures)
if err != nil {
    return err
}

// Verify aggregated signature against aggregated public key
aggPK, err := signature.AggregatePublicKeys(publicKeys)
valid := signature.VerifyAggregateBLS(aggPK, aggregated, msg)
```

### ML-DSA (Dilithium) Signatures

ML-DSA provides NIST-standardized post-quantum security:

```go
// Create ML-DSA signer with security level
signer, err := signature.NewMLDSA(signature.SecurityLevelHigh) // ML-DSA-87

// Available security levels:
// SecurityLevelLow    -> ML-DSA-44 (NIST Level 2)
// SecurityLevelMedium -> ML-DSA-65 (NIST Level 3) - Default
// SecurityLevelHigh   -> ML-DSA-87 (NIST Level 5)

// Sign message
sig, err := signer.Sign(msg)

// Verify signature
valid := signer.Verify(msg, sig)
```

### Hybrid Signatures (BLS + ML-DSA)

Hybrid signatures provide both classical and quantum security:

```go
// Create hybrid signer
signer, err := signature.NewHybrid(signature.HybridConfig{
    BLSWeight:      0.3, // 30% weight for BLS
    QuantumWeight:  0.7, // 70% weight for ML-DSA
    SecurityLevel:  signature.SecurityLevelMedium,
})

// Hybrid sign - creates both BLS and ML-DSA signatures
hybridSig, err := signer.Sign(msg)

// Verify both signatures
valid := signer.Verify(msg, hybridSig)
```

### Corona (Lattice-Based Threshold)

Corona provides post-quantum threshold signatures:

```go
// Create Corona threshold signer
// Parameters: threshold (t), total signers (n)
signer, err := signature.NewCorona(signature.CoronaConfig{
    Threshold:     4,   // Minimum signers required
    TotalSigners:  7,   // Total signers in the group
    SecurityLevel: signature.SecurityLevelMedium,
})

// Generate precomputation for fast signing
precomp, err := signer.Precompute()

// Quick sign using precomputation (sub-millisecond)
share, err := signer.QuickSign(precomp, msg)

// Combine shares when threshold is reached
combined, err := signer.CombineShares(shares)

// Verify threshold signature
valid := signer.Verify(msg, combined)
```

## Consensus Engine Integration

### BFTConfig

Configure the BFT consensus engine with quantum-safe options:

```go
import "github.com/luxfi/sdk"

config := &sdk.BFTConfig{
    // Deployment type
    DeploymentType: sdk.SovereignL1Deployment,

    // Network configuration
    NetworkID: 7777,
    ChainID:   chainID,
    NodeID:    nodeID,

    // Consensus parameters
    MaxProposalWait:    8 * time.Second,
    MaxRebroadcastWait: 20 * time.Second,

    // Enable quantum-safe mode (Quasar Protocol)
    QuantumSafeMode: true,

    // Enable BLS signature aggregation
    BLSAggregation: true,

    // Enable replication protocol
    ReplicationEnabled: true,

    // Validators
    Validators: validatorNodeIDs,

    // Storage
    DB:  database,
    WAL: writeAheadLog,

    // Logging
    Logger: logger,
}

engine, err := sdk.NewBFTConsensusEngine(config)
if err != nil {
    return err
}
```

### Quasar Core

Use the Quasar core for multi-chain quantum consensus:

```go
import "github.com/luxfi/consensus/protocol/quasar"

// Create Quasar consensus with threshold
core, err := quasar.NewQuasar(threshold)
if err != nil {
    return err
}

// Start consensus engine
ctx := context.Background()
if err := core.Start(ctx); err != nil {
    return err
}

// Register additional chains (auto-registers P, X, C chains)
core.RegisterChain("my-subnet")

// Submit blocks for quantum finality
block := &quasar.Block{
    ID:        blockID,
    ChainName: "C-Chain",
    Height:    height,
    Timestamp: time.Now(),
    Data:      blockData,
}
if err := core.SubmitBlock(block); err != nil {
    return err
}

// Check quantum finality
if core.VerifyQuantumFinality(blockHash) {
    fmt.Println("Block has quantum finality")
}
```

## Performance Optimizations

### Object Pools

Use object pools to reduce GC pressure in high-throughput scenarios:

```go
import "github.com/luxfi/sdk/consensus/pool"

// Create signature pool
sigPool := pool.NewSignaturePool(pool.Config{
    InitialSize: 1000,
    MaxSize:     10000,
})

// Get signature from pool
sig := sigPool.Get()
defer sigPool.Put(sig)

// Use signature...
```

### Pre-allocation

Pre-allocate buffers for batch operations:

```go
// Pre-allocate verification batch
batch := signature.NewVerificationBatch(1000)

// Add signatures to batch
for _, sig := range signatures {
    batch.Add(msg, sig, publicKey)
}

// Batch verify (parallelized internally)
results := batch.VerifyAll()
```

### Parallel Verification

Enable parallel signature verification:

```go
// Configure parallelism
signature.SetParallelism(runtime.NumCPU())

// Batch verify with parallelism
valid := signature.ParallelVerify(messages, signatures, publicKeys)
```

## Security Levels

| Level | Classical Security | Quantum Security | ML-DSA Mode | Use Case |
|-------|-------------------|------------------|-------------|----------|
| Low | 128-bit | 64-bit | ML-DSA-44 | Development/Testing |
| Medium | 192-bit | 96-bit | ML-DSA-65 | Production (default) |
| High | 256-bit | 128-bit | ML-DSA-87 | High-security applications |

## Consensus Parameters

| Parameter | Mainnet | Testnet | Description |
|-----------|---------|---------|-------------|
| k | 20 | 15 | Committee sample size |
| alpha | 15 | 11 | Quorum threshold (2k/3 + 1) |
| beta | 4 | 3 | Confidence threshold |
| Round1Timeout | 150ms | 100ms | BLS aggregation timeout |
| Round2Timeout | 350ms | 200ms | Lattice signature timeout |

## Example: Full Consensus Flow

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/luxfi/sdk"
    "github.com/luxfi/sdk/consensus/signature"
    "github.com/luxfi/consensus/protocol/quasar"
)

func main() {
    // 1. Create signature providers
    blsSigner, _ := signature.NewBLS()
    pqSigner, _ := signature.NewMLDSA(signature.SecurityLevelMedium)

    // 2. Create hybrid consensus
    hybrid, _ := quasar.NewHybrid(4) // threshold of 4

    // 3. Add validators with both key types
    for i := 0; i < 7; i++ {
        validatorID := fmt.Sprintf("validator-%d", i)
        hybrid.AddValidator(validatorID, 100)
    }

    // 4. Sign message with hybrid signatures
    msg := []byte("block-hash-12345")
    hybridSig, _ := hybrid.SignMessage("validator-0", msg)

    // 5. Verify hybrid signature
    valid := hybrid.VerifyHybridSignature(msg, hybridSig)
    fmt.Printf("Hybrid signature valid: %v\n", valid)

    // 6. Collect and aggregate signatures
    var signatures []*quasar.HybridSignature
    for i := 0; i < 5; i++ {
        sig, _ := hybrid.SignMessage(fmt.Sprintf("validator-%d", i), msg)
        signatures = append(signatures, sig)
    }

    // 7. Create aggregated signature
    aggSig, _ := hybrid.AggregateSignatures(msg, signatures)
    fmt.Printf("Aggregated %d signatures\n", aggSig.SignerCount)

    // 8. Verify aggregated signature
    validAgg := hybrid.VerifyAggregatedSignature(msg, aggSig)
    fmt.Printf("Aggregated signature valid: %v\n", validAgg)
}
```

## API Reference

See the full API documentation:

- [Signature Interface](/sdk/consensus/signature/)
- [ThresholdSigner Interface](/sdk/consensus/threshold/)
- [BFT Engine](/sdk/bft_engine/)
- [Quasar Protocol](/consensus/protocol/quasar/)

## Related Documentation

- [LP-4110: Quasar Consensus Protocol](/lps/LPs/lp-4110-quasar-consensus-protocol.md)
- [LP-7324: Corona Threshold Signature Precompile](/lps/LPs/lp-7324-corona-threshold-signature-precompile.md)
- [Consensus Architecture](/docs/MULTI_CONSENSUS_ARCHITECTURE.md)
