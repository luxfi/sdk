// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package metrics

import (
	"errors"
	"time"

	"github.com/luxfi/metric"

	"github.com/luxfi/ids"
	utilmetric "github.com/luxfi/sdk/utils/metric"
	"github.com/luxfi/sdk/utils/wrappers"
	"github.com/luxfi/sdk/node/vms/components/gas"
	"github.com/luxfi/sdk/node/vms/platformvm/block"
)

const (
	ResourceLabel   = "resource"
	GasLabel        = "gas"
	ValidatorsLabel = "validators"
)

var (
	gasLabels = metric.Labels{
		ResourceLabel: GasLabel,
	}
	validatorsLabels = metric.Labels{
		ResourceLabel: ValidatorsLabel,
	}
)

var _ Metrics = (*metricsImpl)(nil)

type Block struct {
	Block block.Block

	GasConsumed gas.Gas
	GasState    gas.State
	GasPrice    gas.Price

	ActiveL1Validators   int
	ValidatorExcess      gas.Gas
	ValidatorPrice       gas.Price
	AccruedValidatorFees uint64
}

type Metrics interface {
	utilmetric.APIInterceptor

	// Mark that the given block was accepted.
	MarkAccepted(Block) error

	// Mark that a validator set was created.
	IncValidatorSetsCreated()
	// Mark that a validator set was cached.
	IncValidatorSetsCached()
	// Mark that we spent the given time computing validator diffs.
	AddValidatorSetsDuration(time.Duration)
	// Mark that we computed a validator diff at a height with the given
	// difference from the top.
	AddValidatorSetsHeightDiff(uint64)

	// Mark that this much stake is staked on the node.
	SetLocalStake(uint64)
	// Mark that this much stake is staked in the network.
	SetTotalStake(uint64)
	// Mark when this node will unstake from the Primary Network.
	SetTimeUntilUnstake(time.Duration)
	// Mark when this node will unstake from a net.
	SetTimeUntilNetUnstake(netID ids.ID, timeUntilUnstake time.Duration)
}

func New(registerer metric.Registerer) (Metrics, error) {
	blockMetrics, err := newBlockMetrics(registerer)
	m := &metricsImpl{
		blockMetrics: blockMetrics,
		timeUntilUnstake: metric.NewGauge("time_until_unstake", "Time (in ns) until this node leaves the Primary Network's validator set"),
		timeUntilNetUnstake: metric.NewGauge("time_until_unstake_net", "Time (in ns) until this node leaves the net's validator set"),
		localStake: metric.NewGauge("local_staked", "Amount (in nLUX) of LUX staked on this node"),
		totalStake: metric.NewGauge("total_staked", "Amount (in nLUX) of LUX staked on the Primary Network"),

		gasConsumed: metric.NewCounter("gas_consumed", "Cumulative amount of gas consumed by transactions"),
		gasCapacity: metric.NewGauge("gas_capacity", "Minimum amount of gas that can be consumed in the next block"),
		activeL1Validators: metric.NewGauge("active_l1_validators", "Number of active L1 validators"),
		excess: metric.NewGauge("excess", "Excess usage of a resource over the target usage"),
		price: metric.NewGauge("price", "Price (in nLUX) of a resource"),
		accruedValidatorFees: metric.NewGauge("accrued_validator_fees", "The total cost of running an active L1 validator since Etna activation"),

		validatorSetsCached: metric.NewCounter("validator_sets_cached", "Total number of validator sets cached"),
		validatorSetsCreated: metric.NewCounter("validator_sets_created", "Total number of validator sets created from applying difflayers"),
		validatorSetsHeightDiff: metric.NewGauge("validator_sets_height_diff_sum", "Total number of validator sets diffs applied for generating validator sets"),
		validatorSetsDuration: metric.NewGauge("validator_sets_duration_sum", "Total amount of time generating validator sets in nanoseconds"),
	}

	errs := wrappers.Errs{Err: err}
	registry, ok := registerer.(metric.Registry)
	if !ok {
		return nil, errors.New("registerer must be a Registry")
	}
	apiRequestMetrics, err := utilmetric.NewAPIInterceptor(registry)
	errs.Add(err)
	m.APIInterceptor = apiRequestMetrics

	return m, errs.Err
}

type metricsImpl struct {
	utilmetric.APIInterceptor

	blockMetrics *blockMetrics

	// Staking metrics
	timeUntilUnstake     metric.Gauge
	timeUntilNetUnstake  metric.GaugeVec
	localStake           metric.Gauge
	totalStake           metric.Gauge

	gasConsumed          metric.Counter
	gasCapacity          metric.Gauge
	activeL1Validators   metric.Gauge
	excess               metric.GaugeVec
	price                metric.GaugeVec
	accruedValidatorFees metric.Gauge

	// Validator set diff metrics
	validatorSetsCached     metric.Counter
	validatorSetsCreated    metric.Counter
	validatorSetsHeightDiff metric.Gauge
	validatorSetsDuration   metric.Gauge
}

func (m *metricsImpl) MarkAccepted(b Block) error {
	m.gasConsumed.Add(float64(b.GasConsumed))
	m.gasCapacity.Set(float64(b.GasState.Capacity))
	m.excess.With(gasLabels).Set(float64(b.GasState.Excess))
	m.price.With(gasLabels).Set(float64(b.GasPrice))

	m.activeL1Validators.Set(float64(b.ActiveL1Validators))
	m.excess.With(validatorsLabels).Set(float64(b.ValidatorExcess))
	m.price.With(validatorsLabels).Set(float64(b.ValidatorPrice))
	m.accruedValidatorFees.Set(float64(b.AccruedValidatorFees))

	return b.Block.Visit(m.blockMetrics)
}

func (m *metricsImpl) IncValidatorSetsCreated() {
	m.validatorSetsCreated.Inc()
}

func (m *metricsImpl) IncValidatorSetsCached() {
	m.validatorSetsCached.Inc()
}

func (m *metricsImpl) AddValidatorSetsDuration(d time.Duration) {
	m.validatorSetsDuration.Add(float64(d))
}

func (m *metricsImpl) AddValidatorSetsHeightDiff(d uint64) {
	m.validatorSetsHeightDiff.Add(float64(d))
}

func (m *metricsImpl) SetLocalStake(s uint64) {
	m.localStake.Set(float64(s))
}

func (m *metricsImpl) SetTotalStake(s uint64) {
	m.totalStake.Set(float64(s))
}

func (m *metricsImpl) SetTimeUntilUnstake(timeUntilUnstake time.Duration) {
	m.timeUntilUnstake.Set(float64(timeUntilUnstake))
}

func (m *metricsImpl) SetTimeUntilNetUnstake(netID ids.ID, timeUntilUnstake time.Duration) {
	m.timeUntilNetUnstake.WithLabelValues(netID.String()).Set(float64(timeUntilUnstake))
}
