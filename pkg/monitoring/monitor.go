package monitoring

import (
	"context"

	"github.com/pkg/errors"

	relayLogger "github.com/smartcontractkit/chainlink-relay/pkg/logger"
	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
)

func NewCosmosMonitor(
	ctx context.Context,
	cosmosConfig CosmosConfig,
	l relayLogger.Logger,
) (*relayMonitoring.Monitor, error) {
	chainReader, err := NewThrottledChainReader(cosmosConfig, l)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create throttled chain reader")
	}

	envelopeSourceFactory := NewEnvelopeSourceFactory(
		chainReader,
		relayLogger.With(l, "component", "source-envelope"),
	)
	txResultsFactory := NewTxResultsSourceFactory(
		chainReader,
	)

	monitor, err := relayMonitoring.NewMonitor(
		ctx,
		l,
		cosmosConfig,
		envelopeSourceFactory,
		txResultsFactory,
		CosmosFeedsParser,
		CosmosNodesParser,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to build monitor")
	}

	proxySourceFactory := NewProxySourceFactory(
		chainReader,
		relayLogger.With(l, "component", "source-proxy"),
	)
	monitor.SourceFactories = append(monitor.SourceFactories, proxySourceFactory)

	prometheusExporterFactory := NewPrometheusExporterFactory(
		relayLogger.With(l, "component", "cosmos-prometheus-exporter"),
		NewMetrics(relayLogger.With(l, "component", "cosmos-metrics")),
	)
	monitor.ExporterFactories = append(monitor.ExporterFactories, prometheusExporterFactory)

	return monitor, nil
}
