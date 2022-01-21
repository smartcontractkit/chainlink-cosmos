package main

import (
	"context"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"github.com/smartcontractkit/chainlink-terra/pkg/monitoring"
	"github.com/smartcontractkit/chainlink/core/logger"
)

func main() {
	ctx := context.Background()

	log := logger.NewLogger().With("project", "terra")

	terraConfig, err := monitoring.ParseTerraConfig()
	if err != nil {
		log.Fatalw("failed to parse terra specific configuration", "error", err)
	}

	terraSourceFactory := monitoring.NewTerraSourceFactory(
		log.With("component", "source"),
	)

	relayMonitoring.Facade(
		ctx,
		logWrapper{log},
		terraConfig,
		terraSourceFactory,
		monitoring.TerraFeedParser,
	)

	log.Info("monitor stopped")
}

type logWrapper struct {
	logger.Logger
}

func (l logWrapper) Criticalw(format string, values ...interface{}) {
	l.Logger.CriticalW(format, values...)
}

func (l logWrapper) With(values ...interface{}) relayMonitoring.Logger {
	return logWrapper{l.Logger.With(values...)}
}
