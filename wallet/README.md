# Lux SDK Wallet

Unified wallet implementation for all Lux chains (P, X, C, EVM) with support for both classical and post-quantum cryptography.

## Architecture

```
wallet/
├── crypto/              # Cryptography abstraction layer
│   ├── interface.go     # Unified Signer interface
│   ├── classic/         # Classic crypto (secp256k1, BLS)
│   └── pqc/            # Post-quantum crypto (ML-DSA, ML-KEM, SLH-DSA)
├── chain/              # Chain-specific implementations
│   ├── p/              # P-Chain (Platform) wallet
│   ├── x/              # X-Chain (Exchange) wallet
│   ├── c/              # C-Chain (Contract) wallet
│   └── evm/            # Generic EVM wallet (for subnets like Zoo)
└── primary/            # Unified wallet managing all chains
```

## Supported Cryptography

### Classic Cryptography
- **SECP256K1** - Ethereum-compatible elliptic curve
- **SECP256R1** - NIST P-256
- **BLS** - BLS12-381 for aggregated signatures

### Post-Quantum Cryptography (NIST Standards)
- **ML-DSA** (Dilithium replacement)
  - ML-DSA-87 (high security)
  - ML-DSA-65 (medium security)
  - ML-DSA-44 (low security)
- **SLH-DSA** (SPHINCS+ replacement)
  - SLH-DSA-256
  - SLH-DSA-192
  - SLH-DSA-128
- **ML-KEM** (Kyber replacement) - For key encapsulation

### Hybrid Schemes
- **SECP256K1 + ML-DSA-87** - Classic + PQC
- **BLS + ML-DSA-87** - Aggregatable + PQC

## Usage

### Basic Wallet Creation

```go
import (
    "github.com/luxfi/sdk/wallet"
    "github.com/luxfi/sdk/wallet/crypto"
)

// Create wallet with classic crypto
factory := crypto.NewFactory()
signer, err := factory.NewSigner(crypto.SECP256K1, privateKey)

// Create wallet with post-quantum crypto
pqcSigner, err := factory.NewSigner(crypto.MLDSA87, privateKey)

// Create multi-chain wallet
w, err := wallet.NewPrimary(uri, signer)
```

### Chain-Specific Operations

```go
// P-Chain operations
pWallet := w.P()
tx, err := pWallet.IssueAddValidatorTx(...)

// X-Chain operations
xWallet := w.X()
tx, err := xWallet.IssueBaseTx(...)

// C-Chain operations
cWallet := w.C()
tx, err := cWallet.IssueImportTx(...)

// EVM operations (Zoo, etc.)
evmWallet := w.EVM(chainID)
tx, err := evmWallet.SendTransaction(...)
```

## Post-Quantum Migration

All wallet operations support PQC transparently:

```go
// Same API, different crypto
classicSigner := crypto.NewSigner(crypto.SECP256K1, key)
pqcSigner := crypto.NewSigner(crypto.MLDSA87, pqKey)

// Both work with same wallet interface
wallet1 := wallet.New(uri, classicSigner)
wallet2 := wallet.New(uri, pqcSigner)
```

## Features

- ✅ Unified interface across all chains
- ✅ Classic and PQC crypto support
- ✅ Hardware wallet compatibility
- ✅ BIP32/BIP44 key derivation
- ✅ Address format compatibility
- ✅ Cross-chain operations
- ✅ Atomic swaps between chains
- ✅ Fee estimation
- ✅ Transaction building and signing
