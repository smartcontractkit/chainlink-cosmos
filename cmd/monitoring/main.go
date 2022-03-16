package main

import (
	"context"
	"fmt"
	"log"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"github.com/smartcontractkit/chainlink/core/logger"

	"github.com/smartcontractkit/chainlink-terra/pkg/monitoring"
	pkgClient "github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
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

	client, err := pkgClient.NewClient(
		terraConfig.ChainID,
		terraConfig.TendermintURL,
		terraConfig.ReadTimeout,
		coreLog,
	)
	if err != nil {
		l.Fatalw("failed to create a terra client", "error", err)
		return
	}
	chainReader := monitoring.NewChainReader(client)

	envelopeSourceFactory := monitoring.NewEnvelopeSourceFactory(
		chainReader,
		l.With("component", "source-envelope"),
	)
	txResultsFactory := monitoring.NewTxResultsSourceFactory(
		l.With("component", "source-txresults"),
	)

	entrypoint, err := relayMonitoring.NewEntrypoint(
		ctx,
		l,
		terraConfig,
		envelopeSourceFactory,
		txResultsFactory,
		monitoring.TerraFeedParser,
	)
	if err != nil {
		l.Fatalw("failed to build entrypoint", "error", err)
		return
	}

	proxySourceFactory := monitoring.NewProxySourceFactory(
		chainReader,
		l.With("component", "source-proxy"),
	)
	if entrypoint.Config.Feature.TestOnlyFakeReaders {
		proxySourceFactory = monitoring.NewFakeProxySourceFactory(l.With("component", "fake-proxy-source"))
	}
	entrypoint.SourceFactories = append(entrypoint.SourceFactories, proxySourceFactory)

	prometheusExporterFactory := monitoring.NewPrometheusExporterFactory(
		l.With("component", "terra-prometheus-exporter"),
		monitoring.NewMetrics(l.With("component", "terra-metrics")),
	)
	entrypoint.ExporterFactories = append(entrypoint.ExporterFactories, prometheusExporterFactory)

	entrypoint.Run()
	l.Info("monitor stopped")
}

// adapt core/logger.Logger to monitoring logger.

type logWrapper struct {
	logger.Logger
}

func (l logWrapper) With(values ...interface{}) relayMonitoring.Logger {
	return logWrapper{l.Logger.With(values...)}
}
