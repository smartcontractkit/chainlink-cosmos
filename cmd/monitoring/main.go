package main

import (
	// "context"

	// relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	// "github.com/smartcontractkit/chainlink-terra/pkg/monitoring"
	"github.com/smartcontractkit/chainlink/core/logger"
)

func main() {
	// ctx := context.Background()

	log := logger.NewLogger().With("project", "terra")

	// terraConfig, err := monitoring.ParseTerraConfig()
	// if err != nil {
	// 	log.Fatalw("failed to parse terra specific configuration", "error", err)
	// }

	// terraSourceFactory, err := monitoring.NewTerraSourceFactory(terraConfig, log.With("component", "source"))
	// if err != nil {
	// 	log.Fatalw("failed to initialize Terra source", "error", err)
	// }

	// relayMonitoring.Entrypoint(
	// 	ctx,
	// 	log,
	// 	terraConfig,
	// 	terraSourceFactory,
	// 	monitoring.TerraFeedParser,
	// 	[]relayMonitoring.SourceFactory{},
	// 	[]relayMonitoring.ExporterFactory{},
	// )

	log.Info("monitor stopped")
}
