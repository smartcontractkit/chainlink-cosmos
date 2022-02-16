package main

import (
	"context"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"github.com/smartcontractkit/chainlink-terra/pkg/monitoring"
	pkgClient "github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
	"github.com/smartcontractkit/chainlink/core/logger"
)

func main() {
	ctx := context.Background()

	log := logger.NewLogger().With("project", "terra")

	terraConfig, err := monitoring.ParseTerraConfig()
	if err != nil {
		log.Fatalw("failed to parse terra specific configuration", "error", err)
		return
	}

	client, err := pkgClient.NewClient(
		terraConfig.ChainID,
		terraConfig.TendermintURL,
		terraConfig.ReadTimeout,
		log,
	)
	if err != nil {
		log.Fatalw("failed to create a terra client", "error", err)
		return
	}

	envelopeSourceFactory := monitoring.NewEnvelopeSourceFactory(
		client,
		log.With("component", "source-envelope"),
	)
	txResultsFactory := monitoring.NewTxResultsSourceFactory(
		log.With("component", "source-txresults"),
	)

	entrypoint, err := relayMonitoring.NewEntrypoint(
		ctx,
		logWrapper{log},
		terraConfig,
		envelopeSourceFactory,
		txResultsFactory,
		monitoring.TerraFeedParser,
	)
	if err != nil {
		log.Fatalw("failed to build entrypoint", "error", err)
		return
	}

	entrypoint.Run()
	log.Info("monitor stopped")
}

// adapt core/logger.Logger to monitoring logger.

type logWrapper struct {
	logger.Logger
}

func (l logWrapper) With(values ...interface{}) relayMonitoring.Logger {
	return logWrapper{l.Logger.With(values...)}
}
