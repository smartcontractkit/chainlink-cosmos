package cosmos

import (
	"context"
	"errors"
	"fmt"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	relaytypes "github.com/smartcontractkit/chainlink-relay/pkg/types"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/cosmwasm"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/params"
)

// ErrMsgUnsupported is returned when an unsupported type of message is encountered.
type ErrMsgUnsupported struct {
	Msg cosmosSDK.Msg
}

func (e *ErrMsgUnsupported) Error() string {
	return fmt.Sprintf("unsupported message type %T: %s", e.Msg, e.Msg)
}

var _ relaytypes.Relayer = &Relayer{}

type Relayer struct {
	lggr   logger.Logger
	chain  adapters.Chain
	ctx    context.Context
	cancel func()
}

// Note: constructed in core
func NewRelayer(lggr logger.Logger, chain adapters.Chain) *Relayer {
	ctx, cancel := context.WithCancel(context.Background())

	params.InitCosmosSdk(
		chain.Config().Bech32Prefix(),
		chain.Config().FeeToken(),
	)

	return &Relayer{
		lggr:   lggr,
		chain:  chain,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (r *Relayer) Name() string {
	return r.lggr.Name()
}

// Start starts the relayer respecting the given context.
func (r *Relayer) Start(context.Context) error {
	if r.chain == nil {
		return errors.New("Cosmos unavailable")
	}
	return nil
}

func (r *Relayer) Close() error {
	r.cancel()
	return nil
}

func (r *Relayer) Ready() error {
	return r.chain.Ready()
}

func (r *Relayer) HealthReport() map[string]error {
	return r.chain.HealthReport()
}

func (r *Relayer) NewMercuryProvider(rargs relaytypes.RelayArgs, pargs relaytypes.PluginArgs) (relaytypes.MercuryProvider, error) {
	return nil, errors.New("mercury is not supported for cosmos")
}

func (r *Relayer) NewFunctionsProvider(rargs relaytypes.RelayArgs, pargs relaytypes.PluginArgs) (relaytypes.FunctionsProvider, error) {
	return nil, errors.New("functions are not supported for cosmos")
}

func (r *Relayer) NewConfigProvider(args relaytypes.RelayArgs) (relaytypes.ConfigProvider, error) {
	configProvider, err := cosmwasm.NewConfigProvider(r.ctx, r.lggr, r.chain, args)
	if err != nil {
		// Never return (*configProvider)(nil)
		return nil, err
	}
	return configProvider, err
}

func (r *Relayer) NewMedianProvider(rargs relaytypes.RelayArgs, pargs relaytypes.PluginArgs) (relaytypes.MedianProvider, error) {
	configProvider, err := cosmwasm.NewMedianProvider(r.ctx, r.lggr, r.chain, rargs, pargs)
	if err != nil {
		return nil, err
	}
	return configProvider, err
}
