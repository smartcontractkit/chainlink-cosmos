package cosmos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	relaytypes "github.com/smartcontractkit/chainlink-relay/pkg/types"
	"github.com/smartcontractkit/chainlink-relay/pkg/utils"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

// ErrMsgUnsupported is returned when an unsupported type of message is encountered.
type ErrMsgUnsupported struct {
	Msg cosmosSDK.Msg
}

func (e *ErrMsgUnsupported) Error() string {
	return fmt.Sprintf("unsupported message type %T: %s", e.Msg, e.Msg)
}

type MsgEnqueuer interface {
	// Enqueue enqueues msg for broadcast and returns its id.
	// Returns ErrMsgUnsupported for unsupported message types.
	Enqueue(contractID string, msg cosmosSDK.Msg) (int64, error)
}

// TxManager manages txs composed of batches of queued messages.
type TxManager interface {
	MsgEnqueuer

	// GetMsgs returns any messages matching ids.
	GetMsgs(ids ...int64) (Msgs, error)
	// GasPrice returns the gas price in ucosm.
	GasPrice() (cosmosSDK.DecCoin, error)
}

// CL Core OCR2 job spec RelayConfig member for Terra
type RelayConfig struct {
	ChainID  string `json:"chainID"`  // required
	NodeName string `json:"nodeName"` // optional, defaults to a random node with ChainID
}

var _ relaytypes.Relayer = &Relayer{}

type Relayer struct {
	lggr     logger.Logger
	chainSet ChainSet
	ctx      context.Context
	cancel   func()
}

// Note: constructed in core
func NewRelayer(lggr logger.Logger, chainSet ChainSet) *Relayer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Relayer{
		lggr:     lggr,
		chainSet: chainSet,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the relayer respecting the given context.
func (r *Relayer) Start(context.Context) error {
	if r.chainSet == nil {
		return errors.New("Terra unavailable")
	}
	return nil
}

func (r *Relayer) Close() error {
	r.cancel()
	return nil
}

func (r *Relayer) Ready() error {
	return r.chainSet.Ready()
}

// Healthy only if all subservices are healthy
func (r *Relayer) Healthy() error {
	return r.chainSet.Healthy()
}

func (r *Relayer) NewConfigProvider(args relaytypes.RelayArgs) (relaytypes.ConfigProvider, error) {
	configProvider, err := newConfigProvider(r.ctx, r.lggr, r.chainSet, args)
	if err != nil {
		// Never return (*configProvider)(nil)
		return nil, err
	}
	return configProvider, err
}

func (r *Relayer) NewMedianProvider(rargs relaytypes.RelayArgs, pargs relaytypes.PluginArgs) (relaytypes.MedianProvider, error) {
	configProvider, err := newConfigProvider(r.ctx, r.lggr, r.chainSet, rargs)
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
			r.lggr,
			configProvider.chain.Config(),
		),
	}, nil
}

var _ relaytypes.ConfigProvider = &configProvider{}

type configProvider struct {
	utils.StartStopOnce
	digester    types.OffchainConfigDigester
	reportCodec median.ReportCodec
	lggr        logger.Logger

	tracker     types.ContractConfigTracker
	transmitter types.ContractTransmitter

	chain         Chain
	contractCache *ContractCache
	reader        *OCR2Reader
	contractAddr  cosmosSDK.AccAddress
}

func newConfigProvider(ctx context.Context, lggr logger.Logger, chainSet ChainSet, args relaytypes.RelayArgs) (*configProvider, error) {
	var relayConfig RelayConfig
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

// Start starts OCR2Provider respecting the given context.
func (p *configProvider) Start(context.Context) error {
	return p.StartOnce("TerraRelay", func() error {
		p.lggr.Debugf("Starting")
		return p.contractCache.Start()
	})
}

func (p *configProvider) Close() error {
	return p.StopOnce("TerraRelay", func() error {
		p.lggr.Debugf("Stopping")
		return p.contractCache.Close()
	})
}

func (p *configProvider) ContractConfigTracker() types.ContractConfigTracker {
	return p.tracker
}

func (p *configProvider) OffchainConfigDigester() types.OffchainConfigDigester {
	return p.digester
}

type medianProvider struct {
	*configProvider
	reportCodec median.ReportCodec
	contract    median.MedianContract
	transmitter types.ContractTransmitter
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
