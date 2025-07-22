# ZOO L2 Network

ZOO is a gaming and NFT-focused L2 on Lux.

## Network Details
- **Chain ID**: 200200
- **Native Token**: ZOO
- **Block Time**: ~2 seconds
- **Consensus**: Lux Consensus (Lux L2)


## Key Features
- Optimized for gaming transactions
- Native NFT support
- Low fees for microtransactions
- Cross-chain bridge to Lux C-Chain

## Deployment

### Local Testing
```bash
cd /home/z/work/lux/stack
make deploy-zoo
```

### Mainnet Deployment
```bash
cd /home/z/work/lux/cli
./lux subnet create zoo --evm --genesis ../evm/deployments/zoo/genesis.json
./lux subnet deploy zoo --mainnet --key <your-key>
```

## Configuration
- Gas limit: 15M per block
- Min base fee: 25 GWEI
- Initial allocation: 100M ZOO tokens

## Validators
Contact the Lux team to become a validator for the ZOO subnet.
