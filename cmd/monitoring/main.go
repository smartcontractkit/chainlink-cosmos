package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"

	"github.com/smartcontractkit/chainlink-terra/pkg/monitoring"
	"github.com/smartcontractkit/chainlink-terra/pkg/monitoring/fcdclient"
)

func main() {
	ctx := context.Background()

	l, err := logger.New()
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		if serr := l.Sync(); serr != nil {
			log.Println(fmt.Sprintf("Error while closing Logger: %v", serr))
		}
	}()

	terraConfig, err := monitoring.ParseTerraConfig()
	if err != nil {
		l.Fatalw("failed to parse terra specific configuration", "error", err)
		return
	}

	chainReader := monitoring.NewChainReader(terraConfig, l)
	fcdClient := fcdclient.New(terraConfig.FCDURL, terraConfig.FCDReqsPerSec)

	envelopeSourceFactory := monitoring.NewEnvelopeSourceFactory(
		chainReader,
		fcdClient,
		logger.With(l, "component", "source-envelope"),
	)
	txResultsFactory := monitoring.NewTxResultsSourceFactory(
		fcdClient,
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
		logger.With(l, "component", "source-proxy"),
	)
	monitor.SourceFactories = append(monitor.SourceFactories, proxySourceFactory)

	prometheusExporterFactory := monitoring.NewPrometheusExporterFactory(
		logger.With(l, "component", "terra-prometheus-exporter"),
		monitoring.NewMetrics(logger.With(l, "component", "terra-metrics")),
	)
	monitor.ExporterFactories = append(monitor.ExporterFactories, prometheusExporterFactory)

	monitor.Run()
	l.Info("monitor stopped")
}
