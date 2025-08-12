module github.com/luxfi/sdk/examples/boot-mainnet

go 1.24.5

require (
	github.com/luxfi/log v1.0.4
	github.com/luxfi/sdk v0.0.0
)

require (
	github.com/luxfi/geth v1.16.1 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/luxfi/netrunner-sdk v1.0.0 // indirect
	github.com/onsi/ginkgo/v2 v2.23.4 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/exp v0.0.0-20250718183923-645b1fa84792 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250728155136-f173205681a0 // indirect
	google.golang.org/grpc v1.74.2 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)

replace (
	github.com/luxfi/consensus => ../../../consensus
	github.com/luxfi/crypto => ../../../crypto
	github.com/luxfi/ids => ../../../ids
	github.com/luxfi/log => ../../../log
	github.com/luxfi/metric => ../../../metrics
	github.com/luxfi/netrunner-sdk => ../../../netrunner-sdk
	github.com/luxfi/node => ../../../node
	github.com/luxfi/sdk => ../../
)
