package terra

import (
	"errors"
	"time"

	uuid "github.com/satori/go.uuid"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
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

type TransmissionSigner interface {
	Sign(msg []byte) ([]byte, error)
	PublicKey() cryptotypes.PubKey
}

type OCR2Spec struct {
	ID          int32
	IsBootstrap bool

	// network data
	//NodeEndpointHTTP    string
	//NodeEndpointWS      string
	TendermintRPC       string // URL exposing tendermint RPC (default port is 26657)
	CosmosRPC           string // URL exposing cosmos endpoints (port is 1317, needs to be enabled in terra node config)
	FCDNodeEndpointHTTP string // FCD nodes have /v1/txs/gas_prices
	ChainID             string
	HTTPTimeout         time.Duration

	FallbackGasPrice   string
	GasLimitMultiplier string

	// on-chain data
	ContractID string

	TransmissionSigner TransmissionSigner
}

type Relayer struct {
	lggr Logger
}

// Note: constructed in core
func NewRelayer(lggr Logger) *Relayer {
	return &Relayer{
		lggr: lggr,
	}
}

func (r *Relayer) Start() error {
	// No subservices started on relay start, but when the first job is started
	return nil
}

// Close will close all open subservices
func (r *Relayer) Close() error {
	return nil
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
	var provider ocr2Provider
	spec, ok := s.(OCR2Spec)
	if !ok {
		return provider, errors.New("unsuccessful cast to 'terra.OCR2Spec'")
	}

	client, err := NewClient(spec, r.lggr)
	if err != nil {
		return nil, err
	}
	if err := client.Start(); err != nil {
		return nil, err
	}
	tracker, err := NewContractTracker(spec, externalJobID.String(), &client, r.lggr)
	if err != nil {
		return nil, err
	}

	digester := OffchainConfigDigester{
		ChainID:    spec.ChainID,
		ContractID: spec.ContractID,
	}

	if spec.IsBootstrap {
		// Return early if bootstrap node (doesn't require the full OCR2 provider)
		return ocr2Provider{
			offchainConfigDigester: digester,
			tracker:                tracker,
		}, nil
	}

	reportCodec := ReportCodec{}

	return ocr2Provider{
		client:                 client,
		offchainConfigDigester: digester,
		reportCodec:            reportCodec,
		tracker:                tracker,
		transmitter:            tracker,
		medianContract:         tracker,
	}, nil
}

type ocr2Provider struct {
	client                 Client
	offchainConfigDigester types.OffchainConfigDigester
	reportCodec            median.ReportCodec
	tracker                types.ContractConfigTracker
	transmitter            types.ContractTransmitter
	medianContract         median.MedianContract
}

func (p ocr2Provider) Start() error {
	return nil
}

func (p ocr2Provider) Close() error {
	return p.client.Close()
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
	return p.offchainConfigDigester
}

func (p ocr2Provider) ReportCodec() median.ReportCodec {
	return p.reportCodec
}

func (p ocr2Provider) MedianContract() median.MedianContract {
	return p.medianContract
}
