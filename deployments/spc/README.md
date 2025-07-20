# SPC L2 Network

Sparkle Pony Club (SPC) is a DeFi-focused L2 on Lux.

## Network Details
- **Chain ID**: 36911
- **Native Token**: SPC
- **Block Time**: ~2 seconds
- **Consensus**: Snowball (Lux consensus)

## Key Features
- Optimized for DeFi protocols
- High throughput for trading
- MEV protection
- Cross-chain liquidity bridges

## Deployment

### Local Testing
```bash
cd /home/z/work/lux/stack
make deploy-spc
```

### Mainnet Deployment
```bash
cd /home/z/work/lux/cli
./lux subnet create spc --evm --genesis ../evm/deployments/spc/genesis.json
./lux subnet deploy spc --mainnet --key <your-key>
```

## Configuration
- Gas limit: 20M per block
- Min base fee: 25 GWEI
- Initial allocation: 100M SPC tokens
- Enhanced for high-frequency trading

## Validators
SPC subnet requires validators with high-performance nodes to handle DeFi traffic.