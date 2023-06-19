package cosmwasm

import (
	"context"
	"encoding/json"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	relaytypes "github.com/smartcontractkit/chainlink-relay/pkg/types"
	"github.com/smartcontractkit/chainlink-relay/pkg/utils"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters"
)

var _ relaytypes.ConfigProvider = &configProvider{}

type configProvider struct {
	utils.StartStopOnce
	digester types.OffchainConfigDigester
	lggr     logger.Logger

	tracker types.ContractConfigTracker

	chain         adapters.Chain
	contractCache *ContractCache
	reader        *OCR2Reader
	contractAddr  cosmosSDK.AccAddress
}

// TODO: pass chain instead of chainSet
func NewConfigProvider(ctx context.Context, lggr logger.Logger, chainSet adapters.ChainSet, args relaytypes.RelayArgs) (*configProvider, error) {
	var relayConfig adapters.RelayConfig
	err := json.Unmarshal(args.RelayConfig, &relayConfig)
	if err != nil {
		return nil, err
	}
	contractAddr, err := cosmosSDK.AccAddressFromBech32(args.ContractID)
	if err != nil {
		return nil, err
	}
	chain, err := chainSet.Chain(ctx, relayConfig.ChainID)
	if err != nil {
		return nil, err
	}
	chainReader, err := chain.Reader(relayConfig.NodeName)
	if err != nil {
		return nil, err
	}
	reader := NewOCR2Reader(contractAddr, chainReader, lggr)
	contract := NewContractCache(chain.Config(), reader, lggr)
	tracker := NewContractTracker(chainReader, contract)
	digester := NewOffchainConfigDigester(relayConfig.ChainID, contractAddr)
	return &configProvider{
		digester:      digester,
		tracker:       tracker,
		lggr:          lggr,
		contractCache: contract,
		reader:        reader,
		chain:         chain,
		contractAddr:  contractAddr,
	}, nil
}

func (c *configProvider) Name() string {
	return c.lggr.Name()
}

// Start starts OCR2Provider respecting the given context.
func (c *configProvider) Start(context.Context) error {
	return c.StartOnce("CosmosRelay", func() error {
		c.lggr.Debugf("Starting")
		return c.contractCache.Start()
	})
}

func (c *configProvider) Close() error {
	return c.StopOnce("CosmosRelay", func() error {
		c.lggr.Debugf("Stopping")
		return c.contractCache.Close()
	})
}

func (c *configProvider) HealthReport() map[string]error {
	return map[string]error{c.Name(): c.Healthy()}
}

func (c *configProvider) ContractConfigTracker() types.ContractConfigTracker {
	return c.tracker
}

func (c *configProvider) OffchainConfigDigester() types.OffchainConfigDigester {
	return c.digester
}

type medianProvider struct {
	*configProvider
	reportCodec median.ReportCodec
	contract    median.MedianContract
	transmitter types.ContractTransmitter
}

func NewMedianProvider(ctx context.Context, lggr logger.Logger, chainSet adapters.ChainSet, rargs relaytypes.RelayArgs, pargs relaytypes.PluginArgs) (relaytypes.MedianProvider, error) {
	configProvider, err := NewConfigProvider(ctx, lggr, chainSet, rargs)
	if err != nil {
		return nil, err
	}
	senderAddr, err := cosmosSDK.AccAddressFromBech32(pargs.TransmitterID)
	if err != nil {
		return nil, err
	}

	return &medianProvider{
		configProvider: configProvider,
		reportCodec:    ReportCodec{},
		contract:       configProvider.contractCache,
		transmitter: NewContractTransmitter(
			configProvider.reader,
			rargs.ExternalJobID.String(),
			configProvider.contractAddr,
			senderAddr,
			configProvider.chain.TxManager(),
			lggr,
			configProvider.chain.Config(),
		),
	}, nil
}

func (p *medianProvider) ContractTransmitter() types.ContractTransmitter {
	return p.transmitter
}

func (p *medianProvider) ReportCodec() median.ReportCodec {
	return p.reportCodec
}

func (p *medianProvider) MedianContract() median.MedianContract {
	return p.contractCache
}

func (p *medianProvider) OnchainConfigCodec() median.OnchainConfigCodec {
	return median.StandardOnchainConfigCodec{}
}
