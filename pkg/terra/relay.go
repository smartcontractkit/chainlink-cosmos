package terra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"
	relaytypes "github.com/smartcontractkit/chainlink/core/services/relay/types"
	"github.com/smartcontractkit/chainlink/core/utils"
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

//go:generate mockery --name Logger --output ./mocks/
type Logger interface {
	Tracef(format string, values ...interface{})
	Debugf(format string, values ...interface{})
	Infof(format string, values ...interface{})
	Warnf(format string, values ...interface{})
	Errorf(format string, values ...interface{})
	Criticalf(format string, values ...interface{})
	Panicf(format string, values ...interface{})
	Fatalf(format string, values ...interface{})
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
	// GasPrice returns the gas price in uluna.
	GasPrice() (cosmosSDK.DecCoin, error)
}

// CL Core OCR2 job spec RelayConfig member for Terra
type RelayConfig struct {
	ChainID  string `json:"chainID"`  // required
	NodeName string `json:"nodeName"` // optional, defaults to a random node with ChainID
}

type Relayer struct {
	lggr     Logger
	chainSet ChainSet
	ctx      context.Context
	cancel   func()
}

// Note: constructed in core
func NewRelayer(lggr Logger, chainSet ChainSet) *Relayer {
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

var _ relaytypes.RelayerCtx = &Relayer{}

// NewMedianProvider creates a new OCR2ProviderCtx instance.
func (r *Relayer) NewMedianProvider(args relaytypes.OCR2Args) (relaytypes.MedianProvider, error) {
	relayConfigBytes, err := json.Marshal(args.RelayConfig)
	if err != nil {
		return nil, err
	}
	var relayConfig RelayConfig
	err = json.Unmarshal(relayConfigBytes, &relayConfig)
	if err != nil {
		return nil, err
	}

	chain, err := r.chainSet.Chain(r.ctx, relayConfig.ChainID)
	if err != nil {
		return nil, err
	}
	chainReader, err := chain.Reader(relayConfig.NodeName)
	if err != nil {
		return nil, err
	}
	msgEnqueuer := chain.TxManager()

	contractAddr, err := cosmosSDK.AccAddressFromBech32(args.ContractID)
	if err != nil {
		return nil, err
	}

	reader := NewOCR2Reader(contractAddr, chainReader, r.lggr)
	contract := NewContractCache(chain.Config(), reader, r.lggr)
	tracker := NewContractTracker(chainReader, contract)
	digester := NewOffchainConfigDigester(relayConfig.ChainID, contractAddr)

	if args.IsBootstrap {
		// Return early if bootstrap node (doesn't require the full OCR2 provider)
		return &medianProvider{
			digester:      digester,
			tracker:       tracker,
			lggr:          r.lggr,
			contractCache: contract,
		}, nil
	}

	senderAddr, err := cosmosSDK.AccAddressFromBech32(args.TransmitterID.String)
	if err != nil {
		return nil, err
	}

	reportCodec := ReportCodec{}
	transmitter := NewContractTransmitter(reader, args.ExternalJobID.String(), contractAddr, senderAddr, msgEnqueuer, r.lggr, chain.Config())

	return &medianProvider{
		digester:      digester,
		reportCodec:   reportCodec,
		tracker:       tracker,
		transmitter:   transmitter,
		lggr:          r.lggr,
		contractCache: contract,
	}, nil
}

type medianProvider struct {
	utils.StartStopOnce
	digester    types.OffchainConfigDigester
	reportCodec median.ReportCodec
	lggr        Logger

	tracker     types.ContractConfigTracker
	transmitter types.ContractTransmitter

	contractCache *ContractCache
}

// Start starts OCR2Provider respecting the given context.
func (p *medianProvider) Start(context.Context) error {
	return p.StartOnce("TerraMedianProvider", func() error {
		p.lggr.Debugf("Starting")

		return p.contractCache.Start()
	})
}

func (p *medianProvider) Close() error {
	return p.StopOnce("TerraMedianProvider", func() error {
		p.lggr.Debugf("Stopping")

		return p.contractCache.Close()
	})
}

func (p *medianProvider) ContractTransmitter() types.ContractTransmitter {
	return p.transmitter
}

func (p *medianProvider) ContractConfigTracker() types.ContractConfigTracker {
	return p.tracker
}

func (p *medianProvider) OffchainConfigDigester() types.OffchainConfigDigester {
	return p.digester
}

func (p *medianProvider) ReportCodec() median.ReportCodec {
	return p.reportCodec
}

func (p *medianProvider) MedianContract() median.MedianContract {
	return p.contractCache
}
