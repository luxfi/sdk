module github.com/luxfi/sdk

go 1.24.5

replace (
	github.com/luxfi/crypto => ../crypto
	github.com/luxfi/ids => ../ids
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
	github.com/luxfi/netrunner-sdk v1.0.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/exp v0.0.0-20250718183923-645b1fa84792
)

require (
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/onsi/ginkgo/v2 v2.23.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250728155136-f173205681a0 // indirect
	google.golang.org/grpc v1.74.2 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
