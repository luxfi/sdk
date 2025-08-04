module github.com/luxfi/sdk/examples/boot-mainnet

go 1.22

require github.com/luxfi/sdk v0.0.0

require (
	github.com/ethereum/go-ethereum v1.13.8 // indirect
	github.com/holiman/uint256 v1.2.4 // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
)

replace github.com/luxfi/sdk => ../../
