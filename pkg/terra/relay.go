package terra

import (
	"errors"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"
	uuid "github.com/satori/go.uuid"

	relaytypes "github.com/smartcontractkit/chainlink/core/services/relay/types"
	"github.com/smartcontractkit/chainlink/core/utils"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

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
	Enqueue(contractID string, msg []byte) (int64, error)
	Start() error
	Close() error
}

// CL Core OCR2 job spec RelayConfig member for Terra
type RelayConfig struct {
	ChainID  string `json:"chainID"`  // required
	NodeName string `json:"nodeName"` // optional, defaults to a random node with ChainID
}

type OCR2Spec struct {
	RelayConfig

	ID          int32
	IsBootstrap bool

	// on-chain data
	ContractID    string
	TransmitterID string
}

type Relayer struct {
	lggr     Logger
	chainSet ChainSet
}

// Note: constructed in core
func NewRelayer(lggr Logger, chainSet ChainSet) *Relayer {
	return &Relayer{
		lggr:     lggr,
		chainSet: chainSet,
	}
}

func (r *Relayer) Start() error {
	return r.chainSet.Start()
}

// Close will close all open subservices
func (r *Relayer) Close() error {
	return r.chainSet.Close()
}

func (r *Relayer) Ready() error {
	return r.chainSet.Ready()
}

// Healthy only if all subservices are healthy
func (r *Relayer) Healthy() error {
	return r.chainSet.Healthy()
}

func (r *Relayer) NewOCR2Provider(externalJobID uuid.UUID, s interface{}) (relaytypes.OCR2Provider, error) {
	spec, ok := s.(OCR2Spec)
	if !ok {
		return nil, errors.New("unsuccessful cast to 'terra.OCR2Spec'")
	}

	chain, err := r.chainSet.Chain(spec.ChainID)
	if err != nil {
		return nil, err
	}
	chainReader, err := chain.Reader(spec.NodeName)
	if err != nil {
		return nil, err
	}
	msgEnqueuer := chain.MsgEnqueuer()

	contractAddr, err := cosmosSDK.AccAddressFromBech32(spec.ContractID)
	if err != nil {
		return nil, err
	}

	reader := NewOCR2Reader(contractAddr, chainReader, r.lggr)
	contract := NewContractCache(chain.Config(), reader, r.lggr)
	tracker := NewContractTracker(chainReader, contract)
	digester := NewOffchainConfigDigester(spec.ChainID, contractAddr)

	if spec.IsBootstrap {
		// Return early if bootstrap node (doesn't require the full OCR2 provider)
		return &ocr2Provider{
			digester:      digester,
			tracker:       tracker,
			lggr:          r.lggr,
			contractCache: contract,
		}, nil
	}

	senderAddr, err := cosmosSDK.AccAddressFromBech32(spec.TransmitterID)
	if err != nil {
		return nil, err
	}

	reportCodec := ReportCodec{}
	transmitter := NewContractTransmitter(reader, externalJobID.String(), contractAddr, senderAddr, msgEnqueuer, r.lggr, chain.Config())

	return &ocr2Provider{
		digester:      digester,
		reportCodec:   reportCodec,
		tracker:       tracker,
		transmitter:   transmitter,
		lggr:          r.lggr,
		contractCache: contract,
	}, nil
}

type ocr2Provider struct {
	utils.StartStopOnce
	digester    types.OffchainConfigDigester
	reportCodec median.ReportCodec
	lggr        Logger

	tracker     types.ContractConfigTracker
	transmitter types.ContractTransmitter

	contractCache *ContractCache
}

func (p *ocr2Provider) Start() error {
	return p.StartOnce("TerraOCR2Provider", func() error {
		p.lggr.Debugf("Starting")

		return p.contractCache.Start()
	})
}

func (p *ocr2Provider) Close() error {
	return p.StopOnce("TerraOCR2Provider", func() error {
		p.lggr.Debugf("Stopping")

		return p.contractCache.Close()
	})
}

func (p *ocr2Provider) ContractTransmitter() types.ContractTransmitter {
	return p.transmitter
}

func (p *ocr2Provider) ContractConfigTracker() types.ContractConfigTracker {
	return p.tracker
}

func (p *ocr2Provider) OffchainConfigDigester() types.OffchainConfigDigester {
	return p.digester
}

func (p *ocr2Provider) ReportCodec() median.ReportCodec {
	return p.reportCodec
}

func (p *ocr2Provider) MedianContract() median.MedianContract {
	return p.contractCache
}
