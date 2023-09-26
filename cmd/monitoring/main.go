package main

import (
	"context"
	"log"
	"os"

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

	bech32Prefix := "wasm"
	gasToken := "ucosm"
	if value, isPresent := os.LookupEnv("COSMOS_BECH32_PREFIX"); isPresent {
		bech32Prefix = value
	}
	if value, isPresent := os.LookupEnv("COSMOS_GAS_TOKEN"); isPresent {
		gasToken = value
	}
	// note: need to register bech32 prefix before parsing config to ensure AccAddressFromBech32 returns a correctly prefixed address
	params.InitCosmosSdk(
		bech32Prefix,
		gasToken,
	)

	cosmosConfig, err := monitoring.ParseCosmosConfig()
	if err != nil {
		l.Fatalw("failed to parse cosmos specific configuration", "error", err)
		return
	}

	monitor, err := monitoring.NewCosmosMonitor(ctx, cosmosConfig, l)
	if err != nil {
		l.Fatalw("failed to create new cosmos monitor", "error", err)
		return
	}

	monitor.Run()
	l.Info("monitor stopped")
}
