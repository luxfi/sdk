# Hanzo AI L2 Network

Hanzo is an AI services-focused L2 on Lux.

## Network Details
- **Chain ID**: 36963
- **Native Token**: AI
- **Block Time**: ~2 seconds
- **Consensus**: Snowball (Lux consensus)
- **Total Supply**: 1,000,000,000 AI

## Key Features
- AI compute marketplace
- Model inference endpoints
- Data storage for AI training
- Smart contract AI integrations
- Higher gas limits for compute-intensive operations

## Deployment

### Local Testing
```bash
cd /home/z/work/lux/stack
make deploy-hanzo
```

### Mainnet Deployment
```bash
cd /home/z/work/lux/cli
./lux subnet create hanzo --evm --genesis ../evm/deployments/hanzo/genesis.json
./lux subnet deploy hanzo --mainnet --key <your-key>
```

## Configuration
- Gas limit: 30M per block (higher for AI operations)
- Min base fee: 25 GWEI
- Initial allocation: 1B AI tokens
- Contract deployer allowlist enabled

## Special Features
- Contract deployer allowlist for quality control
- Enhanced gas limits for AI inference
- Native oracles for AI model results

## Validators
Hanzo subnet requires validators with GPU support for AI workload verification.