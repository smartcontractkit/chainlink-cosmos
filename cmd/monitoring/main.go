package main

import (
	"context"
	"fmt"
	"log"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"github.com/smartcontractkit/chainlink/core/logger"

	"github.com/smartcontractkit/chainlink-terra/pkg/monitoring"
)

func main() {
	ctx := context.Background()

	coreLog, closeLggr := logger.NewLogger()
	defer func() {
		if err := closeLggr(); err != nil {
			log.Println(fmt.Sprintf("Error while closing Logger: %v", err))
		}
	}()
	l := logWrapper{coreLog}

	terraConfig, err := monitoring.ParseTerraConfig()
	if err != nil {
		l.Fatalw("failed to parse terra specific configuration", "error", err)
		return
	}

	chainReader := monitoring.NewChainReader(terraConfig, coreLog)

	envelopeSourceFactory := monitoring.NewEnvelopeSourceFactory(
		chainReader,
		l.With("component", "source-envelope"),
	)
	txResultsFactory := monitoring.NewTxResultsSourceFactory(
		l.With("component", "source-txresults"),
	)

	monitor, err := relayMonitoring.NewMonitor(
		ctx,
		l,
		terraConfig,
		envelopeSourceFactory,
		txResultsFactory,
		monitoring.TerraFeedsParser,
		monitoring.TerraNodesParser,
	)
	if err != nil {
		l.Fatalw("failed to build monitor", "error", err)
		return
	}

	proxySourceFactory := monitoring.NewProxySourceFactory(
		chainReader,
		l.With("component", "source-proxy"),
	)
	monitor.SourceFactories = append(monitor.SourceFactories, proxySourceFactory)

	prometheusExporterFactory := monitoring.NewPrometheusExporterFactory(
		l.With("component", "terra-prometheus-exporter"),
		monitoring.NewMetrics(l.With("component", "terra-metrics")),
	)
	monitor.ExporterFactories = append(monitor.ExporterFactories, prometheusExporterFactory)

	monitor.Run()
	l.Info("monitor stopped")
}

// adapt core/logger.Logger to monitoring logger.

type logWrapper struct {
	logger.Logger
}

func (l logWrapper) With(values ...interface{}) relayMonitoring.Logger {
	return logWrapper{l.Logger.With(values...)}
}
