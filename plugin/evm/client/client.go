// (c) 2019-2020, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package client

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/interfaces"
	"github.com/luxfi/evm/plugin/evm/config"
)

// Interface compliance
var _ Client = (*client)(nil)

type CurrentValidator struct {
	ValidationID     interfaces.ID     `json:"validationID"`
	NodeID           interfaces.NodeID `json:"nodeID"`
	Weight           uint64     `json:"weight"`
	StartTimestamp   uint64     `json:"startTimestamp"`
	IsActive         bool       `json:"isActive"`
	IsL1Validator    bool       `json:"isL1Validator"`
	IsConnected      bool       `json:"isConnected"`
	UptimePercentage float32    `json:"uptimePercentage"`
	UptimeSeconds    uint64     `json:"uptimeSeconds"`
}

// Client interface for interacting with EVM [chain]
type Client interface {
	StartCPUProfiler(ctx context.Context, options ...interfaces.RPCOption) error
	StopCPUProfiler(ctx context.Context, options ...interfaces.RPCOption) error
	MemoryProfile(ctx context.Context, options ...interfaces.RPCOption) error
	LockProfile(ctx context.Context, options ...interfaces.RPCOption) error
	SetLogLevel(ctx context.Context, level slog.Level, options ...interfaces.RPCOption) error
	GetVMConfig(ctx context.Context, options ...interfaces.RPCOption) (*interfaces.Config, error)
	GetCurrentValidators(ctx context.Context, nodeIDs []interfaces.NodeID, options ...interfaces.RPCOption) ([]CurrentValidator, error)
}

// Client implementation for interacting with EVM [chain]
type client struct {
	adminRequester      interfaces.EndpointRequester
	validatorsRequester interfaces.EndpointRequester
}

// NewClient returns a Client for interacting with EVM [chain]
func NewClient(uri, chain string) Client {
	requestUri := fmt.Sprintf("%s/ext/bc/%s", uri, chain)
	return NewClientWithURL(requestUri)
}

// NewClientWithURL returns a Client for interacting with EVM [chain]
func NewClientWithURL(url string) Client {
	return &client{
		adminRequester: interfaces.NewEndpointRequester(
			url,
			"admin",
		),
		validatorsRequester: interfaces.NewEndpointRequester(
			url,
			"validators",
		),
	}
}

func (c *client) StartCPUProfiler(ctx context.Context, options ...interfaces.RPCOption) error {
	return c.adminRequester.SendRequest(ctx, "admin.startCPUProfiler", struct{}{}, &api.EmptyReply{}, options...)
}

func (c *client) StopCPUProfiler(ctx context.Context, options ...interfaces.RPCOption) error {
	return c.adminRequester.SendRequest(ctx, "admin.stopCPUProfiler", struct{}{}, &api.EmptyReply{}, options...)
}

func (c *client) MemoryProfile(ctx context.Context, options ...interfaces.RPCOption) error {
	return c.adminRequester.SendRequest(ctx, "admin.memoryProfile", struct{}{}, &api.EmptyReply{}, options...)
}

func (c *client) LockProfile(ctx context.Context, options ...interfaces.RPCOption) error {
	return c.adminRequester.SendRequest(ctx, "admin.lockProfile", struct{}{}, &api.EmptyReply{}, options...)
}

type SetLogLevelArgs struct {
	Level string `json:"level"`
}

// SetLogLevel dynamically sets the log level for the C Chain
func (c *client) SetLogLevel(ctx context.Context, level slog.Level, options ...interfaces.RPCOption) error {
	return c.adminRequester.SendRequest(ctx, "admin.setLogLevel", &SetLogLevelArgs{
		Level: level.String(),
	}, &api.EmptyReply{}, options...)
}

type ConfigReply struct {
	Config *interfaces.Config `json:"config"`
}

// GetVMConfig returns the current config of the VM
func (c *client) GetVMConfig(ctx context.Context, options ...interfaces.RPCOption) (*interfaces.Config, error) {
	res := &ConfigReply{}
	err := c.adminRequester.SendRequest(ctx, "admin.getVMConfig", struct{}{}, res, options...)
	return res.Config, err
}

type GetCurrentValidatorsRequest struct {
	NodeIDs []interfaces.NodeID `json:"nodeIDs"`
}

type GetCurrentValidatorsResponse struct {
	Validators []CurrentValidator `json:"validators"`
}

// GetCurrentValidators returns the current validators
func (c *client) GetCurrentValidators(ctx context.Context, nodeIDs []interfaces.NodeID, options ...interfaces.RPCOption) ([]CurrentValidator, error) {
	res := &GetCurrentValidatorsResponse{}
	err := c.validatorsRequester.SendRequest(ctx, "interfaces.getCurrentValidators", &GetCurrentValidatorsRequest{
		NodeIDs: nodeIDs,
	}, res, options...)
	return res.Validators, err
}
