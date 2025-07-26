# Release Notes

## [v1.13.3](https://github.com/luxfi/evm/releases/tag/v1.13.3)

**First Official Release of Lux EVM**

This is the first official release of Lux EVM, bringing full compatibility with Ethereum Virtual Machine while maintaining our unique features and optimizations.

### Key Features

- Full EVM compatibility for smart contracts and DApps
- Support for Lux mainnet (Chain ID 96369) and testnet (Chain ID 96368)
- POA consensus mode with automining for development environments
- Compatible with ZOO (200200), SPC (36911), and Hanzo (36963/36962) L2 networks
- Built on Go 1.24.5

### Major Components

- **Core EVM Engine**: Full Ethereum Virtual Machine implementation
- **Type System**: Uses Ethereum types throughout for compatibility
- **Adapter Layer**: Proper intermediate layer between node and EVM via geth
- **Consensus**: Our own consensus implementation (replacing snowman)
- **Test Suite**: Full test compatibility ensuring reliability

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

- Extensive test suite ensuring compatibility
- Docker support for easy deployment
- Comprehensive documentation
- Active development and community support