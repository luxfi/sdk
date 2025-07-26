# Release Notes

## Historical Timeline

### Private Testing Phase (January 2021)
- Initial private alpha testing and benchmarking
- Based on early Avalanche EVM implementation
- Used for internal evaluation and performance testing

### Mainnet Launch (January 1, 2022)
- Lux mainnet launched with Chain ID 96369
- Based on subnet-evm v0.2.x (contemporary with avalanchego v1.7.x)
- Initial production deployment

### Second Major Update (January 2023)
- Network upgrade to subnet-evm v0.4.x (contemporary with avalanchego v1.9.x)
- Enhanced stability and performance
- Added new precompiles and features

### Third Major Update (November 2024)
- Upgraded to subnet-evm v0.6.x (contemporary with avalanchego v1.11.x)
- Running stable through July 2025
- Enhanced Warp messaging and cross-chain capabilities

### Current Sync (July 26, 2025)
- Full synchronization with latest upstream versions
- Now at parity with subnet-evm v0.7.7
- All tests passing with upstream compatibility
- Final compatibility build for Lux network

---

## Current Releases

## [v0.8.1](https://github.com/luxfi/evm/releases/tag/v0.8.1)

**Development Release**

Ongoing development work with enhanced features and optimizations.

## [v0.8.0](https://github.com/luxfi/evm/releases/tag/v0.8.0)

**Development Release**

Initial development work for new features.

## [v0.7.7](https://github.com/luxfi/evm/releases/tag/v0.7.7)

**First Official Tagged Release of Lux EVM**

This is the first officially tagged release of Lux EVM, fully synchronized with subnet-evm v0.7.7.

### Key Features

- Full EVM compatibility for smart contracts and DApps
- Support for Lux mainnet (Chain ID 96369) and testnet (Chain ID 96368)
- POA consensus mode with automining for development environments
- Compatible with ZOO (200200), SPC (36911), and Hanzo (36963/36962) L2 networks
- Built with Go 1.24.5
- All upstream tests passing

### Major Enhancements

- **Core EVM Engine**: Full Ethereum Virtual Machine implementation
- **Type System**: Uses Ethereum types throughout for compatibility
- **Adapter Layer**: Proper intermediate layer between node and EVM via geth
- **Consensus**: Our own consensus implementation
- **Test Suite**: Full test compatibility with upstream

### Configuration

- Network ID matches Chain ID for consistency
- Modified consensus parameters for single-node development (snow-sample-size=1, snow-quorum-size=1)
- APIs enabled: eth, web3, admin, debug, personal, txpool, miner
- Test account: 0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC

### Network Support

- **LUX Network**: Primary network with LUX coin
- **ZOO Network**: L2/Subnet with ZOO coin
- **SPC Network**: L2/Subnet with SPC coin  
- **Hanzo Network**: Prepared for AI coin deployment

### Development

- Extensive test suite with full upstream compatibility
- Docker support for easy deployment
- Comprehensive documentation
- Active development and community support