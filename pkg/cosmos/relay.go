package cosmos

import (
	"context"
	"errors"
	"math/big"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-common/pkg/types"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/cosmwasm"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/injective"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/params"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/txm"
)

const (
	InjectivePrefix string = "inj"
)

// ErrMsgUnsupported is returned when an unsupported type of message is encountered.
// Deprecated: use txm.ErrMsgUnsupported
type ErrMsgUnsupported = txm.ErrMsgUnsupported

var _ types.Relayer = &Relayer{} //nolint:staticcheck

type Relayer struct {
	lggr  logger.Logger
	chain adapters.Chain
}

// Note: constructed in core
func NewRelayer(lggr logger.Logger, chain adapters.Chain) *Relayer {
	bech32Prefix := chain.Config().Bech32Prefix()
	gasToken := chain.Config().GasToken()
	params.InitCosmosSdk(
		bech32Prefix,
		gasToken,
	)

	return &Relayer{
		lggr:  logger.Named(lggr, "Relayer"),
		chain: chain,
	}
}

func (r *Relayer) Chain() adapters.Chain { return r.chain }

func (r *Relayer) Name() string {
	return r.lggr.Name()
}

// Start starts the relayer respecting the given context.
func (r *Relayer) Start(ctx context.Context) error {
	if r.chain == nil {
		return errors.New("Cosmos unavailable")
	}
	return r.chain.Start(ctx)
}

func (r *Relayer) Close() error { return r.chain.Close() }

func (r *Relayer) Ready() error {
	return r.chain.Ready()
}

func (r *Relayer) HealthReport() map[string]error {
	hp := map[string]error{r.Name(): nil}
	services.CopyHealth(hp, r.chain.HealthReport())
	return r.chain.HealthReport()
}

func (r *Relayer) GetChainStatus(ctx context.Context) (types.ChainStatus, error) {
	return r.chain.GetChainStatus(ctx)
}

func (r *Relayer) ListNodeStatuses(ctx context.Context, pageSize int32, pageToken string) (stats []types.NodeStatus, nextPageToken string, total int, err error) {
	return r.chain.ListNodeStatuses(ctx, pageSize, pageToken)
}

func (r *Relayer) Transact(ctx context.Context, from, to string, amount *big.Int, balanceCheck bool) error {
	return r.chain.Transact(ctx, from, to, amount, balanceCheck)
}

func (r *Relayer) NewMercuryProvider(rargs types.RelayArgs, pargs types.PluginArgs) (types.MercuryProvider, error) {
	return nil, errors.New("mercury is not supported for cosmos")
}

func (r *Relayer) NewLLOProvider(rargs types.RelayArgs, pargs types.PluginArgs) (types.LLOProvider, error) {
	return nil, errors.New("data streams is not supported for cosmos")
}

func (r *Relayer) NewFunctionsProvider(rargs types.RelayArgs, pargs types.PluginArgs) (types.FunctionsProvider, error) {
	return nil, errors.New("functions are not supported for cosmos")
}

func (r *Relayer) NewConfigProvider(args types.RelayArgs) (types.ConfigProvider, error) {
	var configProvider types.ConfigProvider
	var err error
	if r.chain.Config().Bech32Prefix() == InjectivePrefix {
		configProvider, err = injective.NewConfigProvider(r.lggr, r.chain, args)
	} else {
		// Default to cosmwasm adapter
		configProvider, err = cosmwasm.NewConfigProvider(r.lggr, r.chain, args)
	}
	if err != nil {
		return nil, err
	}

	return configProvider, err
}

func (r *Relayer) NewMedianProvider(rargs types.RelayArgs, pargs types.PluginArgs) (types.MedianProvider, error) {
	configProvider, err := cosmwasm.NewMedianProvider(r.lggr, r.chain, rargs, pargs)
	if err != nil {
		return nil, err
	}
	return configProvider, err
}

func (r *Relayer) NewAutomationProvider(rargs types.RelayArgs, pargs types.PluginArgs) (types.AutomationProvider, error) {
	return nil, errors.New("automation is not supported for cosmos")
}

func (r *Relayer) NewPluginProvider(rargs types.RelayArgs, pargs types.PluginArgs) (types.PluginProvider, error) {
	return nil, errors.New("plugin provider is not supported for cosmos")
}
