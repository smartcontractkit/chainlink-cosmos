package main

import (
	"context"
	"log"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/params"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/monitoring"
	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
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

	params.InitCosmosSdk(
		cosmosConfig.Bech32Prefix,
		cosmosConfig.GasToken,
	)

	monitor, err := monitoring.NewCosmosMonitor(ctx, cosmosConfig, l)
	if err != nil {
		l.Fatalw("failed to create new cosmos monitor", "error", err)
		return
	}

	monitor.Run()
	l.Info("monitor stopped")
}
