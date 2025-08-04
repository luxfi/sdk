# Contributing to sdk

Thank you for your interest in contributing to `sdk`! By contributing to sdk, you are helping to build the foundation for the next generation of blockchains and decentralized applications.

## Getting Started

### Prerequisites

To contribute to `sdk`, you'll need:

- [Go](https://golang.org/dl/) 1.20 or higher

### Setting up your development environment

1. Clone the repository:

```bash
git clone https://github.com/luxfi/sdk.git
cd sdk
```

2. Install the dependencies:

```go
go mod download
```

This will download and install all required dependencies for the project.

## Building and running tests

To build and run tests for the sdk, simply run:

```go
./scripts/tests.unit.sh
```

This will build and run all tests for the project.

## Running linters

To run the linters, simply run:

```go
./scripts/tests.lint.sh
```

This will run the linters on all code in the project.

The `sdk` project also has a fixer that tries to help. To run the fixer, simply run:

```go
./scripts/fix.lint.sh
```

## Contributing

We welcome contributions to sdk! To contribute, please follow these steps:

1. Fork the repository and create a new branch for your contribution.

2. Make your changes and ensure that all tests pass and linting is clean.

3. Write tests for any new features or bug fixes. (If necessary)

4. Submit a pull request with your changes.

## Pull Request Guidelines

When submitting a pull request, please ensure that:

1. Your code is formatted using `go fmt`.

2. Your code is properly tested.

3. Your code passes all linters.

4. Your pull request description explains the problem and solution clearly.
