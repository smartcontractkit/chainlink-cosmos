package injective

import (
	"context"
	"encoding/json"

	tmtypes "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	cosmosSDK "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	relaytypes "github.com/smartcontractkit/chainlink-relay/pkg/types"
	"github.com/smartcontractkit/chainlink-relay/pkg/utils"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/injective/median_report"
	injectivetypes "github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/injective/types"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/client"
)

var _ relaytypes.ConfigProvider = &configProvider{}

type configProvider struct {
	utils.StartStopOnce
	digester types.OffchainConfigDigester
	lggr     logger.Logger

	tracker types.ContractConfigTracker

	chain           adapters.Chain
	reader          client.Reader
	injectiveClient injectivetypes.QueryClient
	feedID          string
}

// TODO: pass chain instead of chainSet
func NewConfigProvider(ctx context.Context, lggr logger.Logger, chainSet adapters.ChainSet, args relaytypes.RelayArgs) (*configProvider, error) {
	var relayConfig adapters.RelayConfig
	err := json.Unmarshal(args.RelayConfig, &relayConfig)
	if err != nil {
		return nil, err
	}
	feedID := args.ContractID // TODO: probably not bech32
	chain, err := chainSet.Chain(ctx, relayConfig.ChainID)
	if err != nil {
		return nil, err
	}
	// TODO: share cosmos.Client or extract the inner clientCtx
	reader, err := chain.Reader(relayConfig.NodeName)
	if err != nil {
		return nil, err
	}
	clientCtx := reader.Context()
	injectiveClient := injectivetypes.NewQueryClient(clientCtx)
	tendermintServiceClient := tmtypes.NewServiceClient(clientCtx)

	tracker := NewCosmosModuleConfigTracker(feedID, injectiveClient, tendermintServiceClient)
	digester := NewCosmosOffchainConfigDigester(relayConfig.ChainID, feedID)
	return &configProvider{
		// TODO:
		digester:        digester,
		tracker:         tracker,
		lggr:            lggr,
		reader:          reader,
		injectiveClient: injectiveClient,
		chain:           chain,
		feedID:          feedID,
	}, nil
}
func (c *configProvider) Name() string {
	return c.lggr.Name()
}

// Start starts OCR2Provider respecting the given context.
func (c *configProvider) Start(context.Context) error {
	return c.StartOnce("CosmosRelay", func() error {
		c.lggr.Debugf("Starting")
		return nil
	})
}

func (c *configProvider) Close() error {
	return c.StopOnce("CosmosRelay", func() error {
		c.lggr.Debugf("Stopping")
		return nil
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

	reportCodec := median_report.ReportCodec{}
	injectiveClient := configProvider.injectiveClient
	contract := NewCosmosMedianReporter(configProvider.feedID, injectiveClient)
	senderAddr, err := cosmosSDK.AccAddressFromBech32(pargs.TransmitterID)
	if err != nil {
		return nil, err
	}
	transmitter := NewCosmosModuleTransmitter(injectiveClient, configProvider.feedID, senderAddr, configProvider.chain.TxManager(), lggr)
	return &medianProvider{
		configProvider: configProvider,
		reportCodec:    reportCodec,
		contract:       contract,
		transmitter:    transmitter,
	}, nil
}

func (p *medianProvider) ContractTransmitter() types.ContractTransmitter {
	return p.transmitter
}

func (p *medianProvider) ReportCodec() median.ReportCodec {
	return p.reportCodec
}

func (p *medianProvider) MedianContract() median.MedianContract {
	return p.contract
}

func (p *medianProvider) OnchainConfigCodec() median.OnchainConfigCodec {
	return median.StandardOnchainConfigCodec{}
}
