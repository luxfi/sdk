module github.com/luxfi/sdk

go 1.24.5

replace (
	github.com/luxfi/crypto => ../crypto
	github.com/luxfi/log => ../log
	github.com/luxfi/metrics => ../metrics
	github.com/luxfi/netrunner-sdk => ../netrunner-sdk
)

require (
	// Core dependencies for working packages
	github.com/btcsuite/btcd/btcutil v1.1.5
	github.com/ethereum/go-ethereum v1.13.8
	github.com/luxfi/crypto v1.2.1
	github.com/luxfi/ids v1.0.2
	github.com/luxfi/log v1.0.0
	github.com/luxfi/metrics v1.0.0
	github.com/luxfi/netrunner-sdk v1.0.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/exp v0.0.0-20240119083558-1b970713d09a
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/holiman/uint256 v1.2.4 // indirect
	github.com/luxfi/crypto v1.1.1 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
