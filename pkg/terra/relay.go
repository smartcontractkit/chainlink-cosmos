package terra

import (
	"errors"
	"time"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/client"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"

	uuid "github.com/satori/go.uuid"

	relaytypes "github.com/smartcontractkit/chainlink/core/services/relay/types"
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

//type TransmissionSigner interface {
//	Sign(msg []byte) ([]byte, error)
//	PublicKey() cryptotypes.PubKey
//}
type MsgEnqueuer interface {
	Enqueue(contractID string, msg []byte) error
	Start() error
	Close() error
}

// CL Core OCR2 job spec RelayConfig member for Terra
type RelayConfig struct {
	// network data
	//TendermintURL string        `json:"tendermintURL"`
	//CosmosURL     string        `json:"cosmosURL"`
	//FcdURL        string        `json:"fcdURL"` // FCD nodes have /v1/txs/gas_prices
	//Timeout       time.Duration `json:"timeout"`
	//ChainID       string        `json:"chainID"`
}

type OCR2Spec struct {
	ID          int32
	IsBootstrap bool

	// network data
	TendermintURL string // URL exposing tendermint RPC (default port is 26657)
	CosmosURL     string // URL exposing cosmos endpoints (port is 1317, needs to be enabled in terra node config)
	FcdURL        string // FCD nodes have /v1/txs/gas_prices
	ChainID       string
	Timeout       time.Duration

	FallbackGasPrice   string
	GasLimitMultiplier string

	// on-chain data
	ContractID    string
	TransmitterID string
}

type Relayer struct {
	lggr Logger
	me   MsgEnqueuer
}

// Note: constructed in core
func NewRelayer(lggr Logger, me MsgEnqueuer) *Relayer {
	return &Relayer{
		lggr: lggr,
		me:   me,
	}
}

func (r *Relayer) Start() error {
	return r.me.Start()
}

// Close will close all open subservices
func (r *Relayer) Close() error {
	return r.me.Close()
}

func (r *Relayer) Ready() error {
	// always ready
	return nil
}

// Healthy only if all subservices are healthy
func (r *Relayer) Healthy() error {
	return nil
}

func (r *Relayer) NewOCR2Provider(externalJobID uuid.UUID, s interface{}) (relaytypes.OCR2Provider, error) {
	spec, ok := s.(OCR2Spec)
	if !ok {
		return nil, errors.New("unsuccessful cast to 'terra.OCR2Spec'")
	}

	contractAddr, err := cosmosSDK.AccAddressFromBech32(spec.ContractID)
	if err != nil {
		return nil, err
	}
	senderAddr, err := cosmosSDK.AccAddressFromBech32(spec.TransmitterID)
	if err != nil {
		return nil, err
	}

	tc, err := client.NewClient(spec.ChainID,
		spec.FallbackGasPrice,
		spec.GasLimitMultiplier,
		spec.TendermintURL,
		spec.CosmosURL,
		spec.FcdURL,
		spec.Timeout,
		r.lggr)
	if err != nil {
		return nil, err
	}
	tracker := NewContractTracker(contractAddr, externalJobID.String(), tc, r.lggr)
	digester := NewOffchainConfigDigester(spec.ChainID, contractAddr)

	if spec.IsBootstrap {
		// Return early if bootstrap node (doesn't require the full OCR2 provider)
		return ocr2Provider{
			digester: digester,
			tracker:  tracker,
		}, nil
	}

	reportCodec := ReportCodec{}
	transmitter := NewContractTransmitter(externalJobID.String(), contractAddr, senderAddr, r.me, tc, r.lggr)
	median := NewMedianContract(contractAddr, tc, r.lggr, transmitter)

	return ocr2Provider{
		digester:       digester,
		reportCodec:    reportCodec,
		tracker:        tracker,
		transmitter:    transmitter,
		medianContract: median,
	}, nil
}

type ocr2Provider struct {
	digester       types.OffchainConfigDigester
	reportCodec    median.ReportCodec
	tracker        types.ContractConfigTracker
	transmitter    types.ContractTransmitter
	medianContract median.MedianContract
}

func (p ocr2Provider) Start() error {
	return nil
}

func (p ocr2Provider) Close() error {
	return nil
}

func (p ocr2Provider) Ready() error {
	// always ready
	return nil
}

func (p ocr2Provider) Healthy() error {
	return nil
}

func (p ocr2Provider) ContractTransmitter() types.ContractTransmitter {
	return p.transmitter
}

func (p ocr2Provider) ContractConfigTracker() types.ContractConfigTracker {
	return p.tracker
}

func (p ocr2Provider) OffchainConfigDigester() types.OffchainConfigDigester {
	return p.digester
}

func (p ocr2Provider) ReportCodec() median.ReportCodec {
	return p.reportCodec
}

func (p ocr2Provider) MedianContract() median.MedianContract {
	return p.medianContract
}
