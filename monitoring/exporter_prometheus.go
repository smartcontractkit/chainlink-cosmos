package monitoring

import (
	"context"
	"fmt"
	"sync"

	relayMonitoring "github.com/smartcontractkit/chainlink-common/pkg/monitoring"
)

// NewPrometheusExporterFactory builds an implementation of the Exporter for prometheus.
func NewPrometheusExporterFactory(
	log relayMonitoring.Logger,
	metrics Metrics,
) relayMonitoring.ExporterFactory {
	return &prometheusExporterFactory{
		log,
		metrics,
	}
}

type prometheusExporterFactory struct {
	log     relayMonitoring.Logger
	metrics Metrics
}

func (p *prometheusExporterFactory) NewExporter(
	params relayMonitoring.ExporterParams,
) (relayMonitoring.Exporter, error) {
	feedConfig, ok := params.FeedConfig.(CosmosFeedConfig)
	if !ok {
		return nil, fmt.Errorf("expected feedConfig to be of type CosmosFeedConfig not %T", params.FeedConfig)
	}
	return &prometheusExporter{
		params.ChainConfig,
		feedConfig,
		p.log,
		p.metrics,
		sync.Mutex{},
		map[string]struct{}{},
	}, nil
}

type prometheusExporter struct {
	chainConfig  relayMonitoring.ChainConfig
	feedConfig   CosmosFeedConfig
	log          relayMonitoring.Logger
	metrics      Metrics
	addressesMu  sync.Mutex
	addressesSet map[string]struct{}
}

func (p *prometheusExporter) Export(ctx context.Context, data interface{}) {
	proxyData, isProxyData := data.(ProxyData)
	if !isProxyData {
		return
	}
	answer := float64(proxyData.Answer.Uint64())
	multiply := float64(p.feedConfig.Multiply.Uint64())
	if multiply == 0 {
		multiply = 1.0
	}
	p.metrics.SetProxyAnswersRaw(
		answer,
		p.feedConfig.ProxyAddressBech32,
		p.feedConfig.GetID(),
		p.chainConfig.GetChainID(),
		p.feedConfig.GetContractStatus(),
		p.feedConfig.GetContractType(),
		p.feedConfig.GetName(),
		p.feedConfig.GetPath(),
		p.chainConfig.GetNetworkID(),
		p.chainConfig.GetNetworkName(),
	)
	p.metrics.SetProxyAnswers(
		answer/multiply,
		p.feedConfig.ProxyAddressBech32,
		p.feedConfig.GetID(),
		p.chainConfig.GetChainID(),
		p.feedConfig.GetContractStatus(),
		p.feedConfig.GetContractType(),
		p.feedConfig.GetName(),
		p.feedConfig.GetPath(),
		p.chainConfig.GetNetworkID(),
		p.chainConfig.GetNetworkName(),
	)
	p.addressesMu.Lock()
	defer p.addressesMu.Unlock()
	p.addressesSet[p.feedConfig.ProxyAddressBech32] = struct{}{}
}

func (p *prometheusExporter) Cleanup(_ context.Context) {
	p.addressesMu.Lock()
	defer p.addressesMu.Unlock()
	for address := range p.addressesSet {
		p.metrics.Cleanup(
			address,
			p.feedConfig.GetContractAddress(),
			p.chainConfig.GetChainID(),
			p.feedConfig.GetContractStatus(),
			p.feedConfig.GetContractType(),
			p.feedConfig.GetName(),
			p.feedConfig.GetPath(),
			p.chainConfig.GetNetworkID(),
			p.chainConfig.GetNetworkName(),
		)
	}
}
