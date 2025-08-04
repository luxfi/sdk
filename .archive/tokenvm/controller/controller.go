// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"context"
	"fmt"

	ametrics "github.com/luxfi/api/metrics"
	"github.com/luxfi/database"
	"github.com/luxfi/sdk/builder"
	"github.com/luxfi/sdk/chain"
	"github.com/luxfi/sdk/gossiper"
	"github.com/luxfi/sdk/pebble"
	"github.com/luxfi/sdk/utils"
	"github.com/luxfi/sdk/vm"
	"github.com/luxfi/snow"
	"github.com/luxfi/snow/engine/common"
	"go.uber.org/zap"

	"github.com/luxfi/sdk/examples/tokenvm/actions"
	"github.com/luxfi/sdk/examples/tokenvm/auth"
	"github.com/luxfi/sdk/examples/tokenvm/config"
	"github.com/luxfi/sdk/examples/tokenvm/consts"
	"github.com/luxfi/sdk/examples/tokenvm/genesis"
	"github.com/luxfi/sdk/examples/tokenvm/storage"
	tutils "github.com/luxfi/sdk/examples/tokenvm/utils"
	"github.com/luxfi/sdk/examples/tokenvm/version"
)

var _ vm.Controller = (*Controller)(nil)

type Controller struct {
	inner *vm.VM

	snowCtx      *snow.Context
	genesis      *genesis.Genesis
	config       *config.Config
	stateManager *StateManager

	metrics *metrics

	metaDB database.Database

	orderBook *OrderBook
}

func New() *vm.VM {
	return vm.New(&Controller{}, version.Version)
}

func (c *Controller) Initialize(
	inner *vm.VM,
	snowCtx *snow.Context,
	gatherer ametrics.MultiGatherer,
	genesisBytes []byte,
	upgradeBytes []byte, // subnets to allow for LWM
	configBytes []byte,
) (
	vm.Config,
	vm.Genesis,
	builder.Builder,
	gossiper.Gossiper,
	database.Database,
	database.Database,
	vm.Handlers,
	chain.ActionRegistry,
	chain.AuthRegistry,
	error,
) {
	c.inner = inner
	c.snowCtx = snowCtx
	c.stateManager = &StateManager{}

	// Instantiate metrics
	var err error
	c.metrics, err = newMetrics(gatherer)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	// Load config and genesis
	c.config, err = config.New(c.snowCtx.NodeID, configBytes)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	c.snowCtx.Log.SetLevel(c.config.GetLogLevel())
	snowCtx.Log.Info("loaded config", zap.Any("contents", c.config))

	c.genesis, err = genesis.New(genesisBytes, upgradeBytes)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf(
			"unable to read genesis: %w",
			err,
		)
	}
	snowCtx.Log.Info("loaded genesis", zap.Any("genesis", c.genesis))

	// Create DBs
	blockPath, err := utils.InitSubDirectory(snowCtx.ChainDataDir, "block")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	// TODO: tune Pebble config based on each sub-db focus
	cfg := pebble.NewDefaultConfig()
	blockDB, err := pebble.New(blockPath, cfg)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	statePath, err := utils.InitSubDirectory(snowCtx.ChainDataDir, "state")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	stateDB, err := pebble.New(statePath, cfg)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	metaPath, err := utils.InitSubDirectory(snowCtx.ChainDataDir, "metadata")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	c.metaDB, err = pebble.New(metaPath, cfg)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	// Create handlers
	apis := map[string]*common.HTTPHandler{}
	endpoint, err := utils.NewHandler(consts.Name, &Handler{inner.Handler(), c})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	apis[vm.Endpoint] = endpoint

	// Create builder and gossiper
	var (
		build  builder.Builder
		gossip gossiper.Gossiper
	)
	if c.config.TestMode {
		c.inner.Logger().Info("running build and gossip in test mode")
		build = builder.NewManual(inner)
		gossip = gossiper.NewManual(inner)
	} else {
		build = builder.NewTime(inner, builder.DefaultTimeConfig())
		gcfg := gossiper.DefaultProposerConfig()
		gcfg.BuildProposerDiff = 1 // don't gossip if producing the next block
		gossip = gossiper.NewProposer(inner, gcfg)
	}

	// Initialize order book used to track all open orders
	c.orderBook = NewOrderBook(c, c.config.TrackedPairs)
	return c.config, c.genesis, build, gossip, blockDB, stateDB, apis, consts.ActionRegistry, consts.AuthRegistry, nil
}

func (c *Controller) Rules(t int64) chain.Rules {
	// TODO: extend with [UpgradeBytes]
	return c.genesis.Rules(t)
}

func (c *Controller) StateManager() chain.StateManager {
	return c.stateManager
}

func (c *Controller) Accepted(ctx context.Context, blk *chain.StatelessBlock) error {
	batch := c.metaDB.NewBatch()
	defer batch.Reset()

	results := blk.Results()
	for i, tx := range blk.Txs {
		result := results[i]
		err := storage.StoreTransaction(
			ctx,
			batch,
			tx.ID(),
			blk.GetTimestamp(),
			result.Success,
			result.Units,
		)
		if err != nil {
			return err
		}
		if result.Success {
			switch action := tx.Action.(type) {
			case *actions.CreateAsset:
				c.metrics.createAsset.Inc()
			case *actions.MintAsset:
				c.metrics.mintAsset.Inc()
			case *actions.BurnAsset:
				c.metrics.burnAsset.Inc()
			case *actions.ModifyAsset:
				c.metrics.modifyAsset.Inc()
			case *actions.Transfer:
				c.metrics.transfer.Inc()
			case *actions.CreateOrder:
				c.metrics.createOrder.Inc()
				actor := auth.GetActor(tx.Auth)
				c.orderBook.Add(
					actions.PairID(action.In, action.Out),
					&Order{
						tx.ID(),
						tutils.Address(actor),
						action.InTick,
						action.OutTick,
						action.Supply,
						actor,
					},
				)
			case *actions.FillOrder:
				c.metrics.fillOrder.Inc()
				orderResult, err := actions.UnmarshalOrderResult(result.Output)
				if err != nil {
					// This should never happen
					return err
				}
				if orderResult.Remaining == 0 {
					c.orderBook.Remove(action.Order)
					continue
				}
				c.orderBook.UpdateRemaining(action.Order, orderResult.Remaining)
			case *actions.CloseOrder:
				c.metrics.closeOrder.Inc()
				c.orderBook.Remove(action.Order)
			case *actions.ImportAsset:
				c.metrics.importAsset.Inc()
			case *actions.ExportAsset:
				c.metrics.exportAsset.Inc()
			}
		}
	}
	return batch.Write()
}

func (*Controller) Rejected(context.Context, *chain.StatelessBlock) error {
	return nil
}

func (*Controller) Shutdown(context.Context) error {
	// Do not close any databases provided during initialization. The VM will
	// close any databases your provided.
	return nil
}
