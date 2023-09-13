package main

import (
	"context"
	"log"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/monitoring"
)

func main() {
	ctx := context.Background()

	l, err := logger.New()
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		if serr := l.Sync(); serr != nil {
			log.Printf("Error while closing Logger: %v\n", serr)
		}
	}()

	cosmosConfig, err := monitoring.ParseCosmosConfig()
	if err != nil {
		l.Fatalw("failed to parse cosmos specific configuration", "error", err)
		return
	}

	chainReader, err := monitoring.NewThrottledChainReader(cosmosConfig, l)
	if err != nil {
		l.Fatalw("Failed to create throttled chain reader", "error", err)
		return
	}

	envelopeSourceFactory := monitoring.NewEnvelopeSourceFactory(
		chainReader,
		logger.With(l, "component", "source-envelope"),
	)
	txResultsFactory := monitoring.NewTxResultsSourceFactory(
		chainReader,
	)

	monitor, err := relayMonitoring.NewMonitor(
		ctx,
		l,
		cosmosConfig,
		envelopeSourceFactory,
		txResultsFactory,
		monitoring.CosmosFeedsParser,
		monitoring.CosmosNodesParser,
	)
	if err != nil {
		l.Fatalw("failed to build monitor", "error", err)
		return
	}

	proxySourceFactory := monitoring.NewProxySourceFactory(
		chainReader,
		logger.With(l, "component", "source-proxy"),
	)
	monitor.SourceFactories = append(monitor.SourceFactories, proxySourceFactory)

	prometheusExporterFactory := monitoring.NewPrometheusExporterFactory(
		logger.With(l, "component", "cosmos-prometheus-exporter"),
		monitoring.NewMetrics(logger.With(l, "component", "cosmos-metrics")),
	)
	monitor.ExporterFactories = append(monitor.ExporterFactories, prometheusExporterFactory)

	monitor.Run()
	l.Info("monitor stopped")
}
