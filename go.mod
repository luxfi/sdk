module github.com/luxfi/sdk

go 1.26.2

exclude github.com/luxfi/geth v1.16.1

require (
	// Core dependencies for working packages
	github.com/btcsuite/btcd/btcutil v1.1.6
	github.com/luxfi/crypto v1.18.3
	github.com/luxfi/geth v1.16.87
	github.com/luxfi/ids v1.2.9
	github.com/luxfi/log v1.4.1
	github.com/luxfi/version v1.0.1
	github.com/luxfi/warp v1.18.5
	github.com/manifoldco/promptui v0.9.0
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	go.uber.org/mock v0.6.0
	go.uber.org/zap v1.27.1
	golang.org/x/crypto v0.50.0
	golang.org/x/exp v0.0.0-20260312153236-7ab1446f8b90 // indirect
	golang.org/x/net v0.52.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

exclude (
	google.golang.org/genproto/googleapis/api v0.0.0-20250721164621-a45f3dfb1074
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250728155136-f173205681a0
)

require (
	connectrpc.com/connect v1.19.1
	github.com/btcsuite/btcutil v1.0.2
	github.com/cavaliergopher/grab/v3 v3.0.1
	github.com/chelnak/ysmrr v0.6.0
	github.com/go-git/go-git/v5 v5.16.2
	github.com/k0kubun/go-ansi v0.0.0-20180517002512-3bf9e2903213
	github.com/luxfi/address v1.0.1
	github.com/luxfi/api v1.0.10
	github.com/luxfi/bft v0.1.5
	github.com/luxfi/consensus v1.22.85
	github.com/luxfi/constants v1.4.7
	github.com/luxfi/database v1.18.3
	github.com/luxfi/evm v0.8.35
	github.com/luxfi/genesis v1.7.1
	github.com/luxfi/go-bip32 v1.0.2
	github.com/luxfi/go-bip39 v1.1.2
	github.com/luxfi/keychain v1.0.2
	github.com/luxfi/ledger v1.1.6
	github.com/luxfi/lpm v1.9.4
	github.com/luxfi/math v1.4.0
	github.com/luxfi/math/safe v0.0.1
	github.com/luxfi/net v0.0.4
	github.com/luxfi/netrunner v1.18.0
	github.com/luxfi/protocol v0.0.3
	github.com/luxfi/rpc v1.0.2
	github.com/luxfi/runtime v1.0.1
	github.com/luxfi/utils v1.1.4
	github.com/luxfi/utxo v0.3.0
	github.com/luxfi/validators v1.0.0
	github.com/melbahja/goph v1.4.0
	github.com/olekukonko/tablewriter v1.0.9
	github.com/schollz/progressbar/v3 v3.18.0
	github.com/spf13/afero v1.15.0
	golang.org/x/mod v0.34.0
	golang.org/x/text v0.36.0
)

require (
	github.com/luxfi/accel v1.0.9 // indirect
	github.com/luxfi/compress v0.0.5 // indirect
	github.com/luxfi/concurrent v0.0.3 // indirect
	github.com/luxfi/container v0.0.4 // indirect
	github.com/luxfi/metric v1.5.1 // indirect
	github.com/luxfi/mock v0.1.1 // indirect
	github.com/luxfi/timer v1.0.2 // indirect
)

require (
	dario.cat/mergo v1.0.0 // indirect
	filippo.io/hpke v0.4.0 // indirect
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.1.6 // indirect
	github.com/StephenButtolph/canoto v0.17.3 // indirect
	github.com/VictoriaMetrics/fastcache v1.13.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.24.4 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.6 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/cockroachdb/errors v1.12.0 // indirect
	github.com/cockroachdb/fifo v0.0.0-20240816210425-c5d0cb0b6fc0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/pebble v1.1.5 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20250429170803-42689b6311bb // indirect
	github.com/consensys/gnark-crypto v0.20.1 // indirect
	github.com/crate-crypto/go-eth-kzg v1.5.0 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240724233137-53bbb0ceb27a // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/deckarep/golang-set/v2 v2.8.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/emicklei/dot v1.11.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/ethereum/c-kzg-4844/v2 v2.1.7 // indirect
	github.com/ethereum/go-verkle v0.2.2 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/ferranbt/fastssz v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/getsentry/sentry-go v0.44.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.2 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/mock v1.7.0-rc.1 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/renameio/v2 v2.0.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/rpc v1.2.1 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/juju/fslock v0.0.0-20160525022230-4d5c94c67b4b // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/luxfi/age v1.5.0 // indirect
	github.com/luxfi/atomic v1.0.0 // indirect
	github.com/luxfi/filesystem v0.0.1 // indirect
	github.com/luxfi/node v1.24.22 // indirect
	github.com/luxfi/trace v0.1.4 // indirect
	github.com/luxfi/zapdb v1.10.0 // indirect
	github.com/luxfi/zwing v0.5.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.21 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.0.100 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pjbgf/sha1cd v0.3.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/sftp v1.13.5 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/skeema/knownhosts v1.3.1 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/supranational/blst v0.3.16 // indirect
	github.com/tinylib/msgp v1.6.1 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.39.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.39.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.42.0 // indirect
	go.opentelemetry.io/otel/sdk v1.40.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/term v0.42.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	gonum.org/v1/gonum v0.17.0 // indirect
	google.golang.org/grpc v1.80.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

require (
	github.com/ProjectZKM/Ziren/crates/go-runtime/zkvm_runtime v0.0.0-20260311194731-d5b7577c683d // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/dgraph-io/badger/v4 v4.9.0 // indirect
	github.com/dgraph-io/ristretto/v2 v2.4.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ethereum/go-bigmodexpfix v0.0.0-20250911101455-f9e208c548ab // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/luxfi/cache v1.2.1 // indirect
	github.com/luxfi/codec v1.1.4
	github.com/luxfi/config v1.1.2 // indirect
	github.com/luxfi/formatting v1.0.1
	github.com/luxfi/hid v0.9.3 // indirect
	github.com/luxfi/keys v1.0.8 // indirect
	github.com/luxfi/math/big v0.1.0 // indirect
	github.com/luxfi/p2p v1.19.2 // indirect
	github.com/luxfi/precompile v0.5.14 // indirect
	github.com/luxfi/sampler v1.0.0 // indirect
	github.com/luxfi/tls v1.0.3
	github.com/luxfi/upgrade v1.0.0
	github.com/luxfi/vm v1.0.40
	github.com/olekukonko/errors v1.1.0 // indirect
	github.com/olekukonko/ll v0.0.9 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20220614013038-64ee5596c38a // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
)

// Use local packages with correct codec (CreateNetworkTx, CreateChainTx)

// Use local sdk/api module for platformvm
