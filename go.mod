module github.com/luxfi/evm

go 1.23.9

replace (
	github.com/luxfi/node => /Users/z/work/lux/node
	github.com/luxfi/geth => /Users/z/work/lux/geth
)

require (
	github.com/VictoriaMetrics/fastcache v1.12.2
	github.com/antithesishq/antithesis-sdk-go v0.4.4
	github.com/cespare/cp v0.1.0
	github.com/cockroachdb/pebble v1.1.5
	github.com/davecgh/go-spew v1.1.1
	github.com/deckarep/golang-set/v2 v2.6.0
	github.com/ethereum/go-ethereum v1.16.1
	github.com/fjl/gencodec v0.1.1
	github.com/go-cmd/cmd v1.4.1
	github.com/gorilla/rpc v1.2.0
	github.com/gorilla/websocket v1.5.0
	github.com/hashicorp/go-bexpr v0.1.10
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/holiman/billy v0.0.0-20240216141850-2abb0c79d3c4
	github.com/holiman/bloomfilter/v2 v2.0.3
	github.com/holiman/uint256 v1.3.2
	github.com/luxfi/node v1.11.10
	github.com/mattn/go-colorable v0.1.13
	github.com/mattn/go-isatty v0.0.20
	github.com/onsi/ginkgo/v2 v2.13.1
	github.com/prometheus/client_golang v1.16.0
	github.com/prometheus/client_model v0.3.0
	github.com/spf13/cast v1.5.0
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.10.0
	github.com/tyler-smith/go-bip39 v1.1.0
	github.com/urfave/cli/v2 v2.27.5
	go.uber.org/goleak v1.3.0
	go.uber.org/mock v0.5.0
	go.uber.org/zap v1.26.0
	golang.org/x/crypto v0.36.0
	golang.org/x/exp v0.0.0-20241215155358-4a5509556b9e
	golang.org/x/mod v0.22.0
	golang.org/x/sync v0.12.0
	golang.org/x/time v0.9.0
	golang.org/x/tools v0.29.0
	google.golang.org/protobuf v1.35.2
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

replace github.com/luxfi/node => /Users/z/work/lux/node