# Lux EVM

[![Build + Test + Release](https://github.com/luxfi/evm/actions/workflows/lint-tests-release.yml/badge.svg)](https://github.com/luxfi/evm/actions/workflows/lint-tests-release.yml)
[![CodeQL](https://github.com/luxfi/evm/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/luxfi/evm/actions/workflows/codeql-analysis.yml)

[Lux](https://docs.lux.network/overview/getting-started/lux-platform) is a network composed of multiple blockchains.
Each blockchain is an instance of a Virtual Machine (VM), much like an object in an object-oriented language is an instance of a class.
That is, the VM defines the behavior of the blockchain.

EVM is the [Virtual Machine (VM)](https://docs.lux.network/learn/lux/virtual-machines) that defines the EVM Contract Chains. Lux EVM is a fully compatible Ethereum Virtual Machine implementation.

This chain implements the Ethereum Virtual Machine and supports Solidity smart contracts as well as most other Ethereum client functionality.

## Building

The EVM runs in a separate process from the main Lux process and communicates with it over a local gRPC connection.

### Lux Compatibility

```text
[v1.13.3] First Official Release - Full EVM compatibility
```

### Building the EVM

To build a binary for the EVM, run the build script:

```bash
./scripts/build.sh
```

This will build the EVM binary and place it in `./build/`.

The binary built by `build.sh` is compatible with Lux Node.

#### Building EVM in Docker

To build a Docker image for the latest EVM version, run:

```bash
./scripts/build_docker_image.sh
```

To build a Docker image from a specific EVM commit or branch, run:

```bash
EVM_COMMIT=<commit_or_branch> ./scripts/build_docker_image.sh
```

### Running the EVM

To start a node with the EVM binary, use:

```bash
./build/luxd --network-id=96369
```

## Testing

### Running Tests

To run all tests, use:

```bash
./scripts/build_test.sh
```

To run a specific test or set of tests, use:

```bash
./scripts/build_test.sh <test_name>
```

## Configuration

### Network Configuration

The EVM supports multiple networks:
- **LUX Mainnet**: Chain ID 96369
- **LUX Testnet**: Chain ID 96368
- **ZOO Network**: Chain ID 200200 (L2/Subnet)
- **SPC Network**: Chain ID 36911 (L2/Subnet)
- **Hanzo Network**: Chain ID 36963 (Prepared for deployment)

### Development Configuration

For local development with POA consensus:
```json
{
  "network-id": 96369,
  "staking-enabled": false,
  "sybil-protection-enabled": false,
  "snow-sample-size": 1,
  "snow-quorum-size": 1
}
```

## Docker

### Build the Docker image

```bash
docker build -t luxfi/evm:latest .
```

### Run the Docker image

```bash
docker run -it luxfi/evm:latest
```

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for more information.

## License

This project is licensed under the LGPL-3.0 License - see the [LICENSE](LICENSE) file for details.