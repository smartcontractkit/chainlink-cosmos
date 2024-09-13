package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/pelletier/go-toml/v2"

	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink-common/pkg/sqlutil"
	"github.com/smartcontractkit/chainlink-common/pkg/types/core"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos"
	coscfg "github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/config"
)

const loggerName = "PluginCosmos"

func main() {
	s := loop.MustNewStartedServer(loggerName)
	defer s.Stop()

	p := &pluginRelayer{Plugin: loop.Plugin{Logger: s.Logger}, ds: s.DataSource}
	defer s.Logger.ErrorIfFn(p.Close, "Failed to close")

	s.MustRegister(p)

	stopCh := make(chan struct{})
	defer close(stopCh)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: loop.PluginRelayerHandshakeConfig(),
		Plugins: map[string]plugin.Plugin{
			loop.PluginRelayerName: &loop.GRPCPluginRelayer{
				PluginServer: p,
				BrokerConfig: loop.BrokerConfig{
					StopCh:   stopCh,
					Logger:   s.Logger,
					GRPCOpts: s.GRPCOpts,
				},
			},
		},
		GRPCServer: s.GRPCOpts.NewServer,
	})
}

type pluginRelayer struct {
	loop.Plugin
	ds sqlutil.DataSource
}

func (c *pluginRelayer) NewRelayer(ctx context.Context, config string, keystore loop.Keystore, capRegistry core.CapabilitiesRegistry) (loop.Relayer, error) {
	d := toml.NewDecoder(strings.NewReader(config))
	d.DisallowUnknownFields()
	var cfg struct {
		Cosmos coscfg.TOMLConfig
	}

	if err := d.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config toml: %w:\n\t%s", err, config)
	}

	opts := cosmos.ChainOpts{
		Logger:   c.Logger,
		KeyStore: keystore,
		DS:       c.ds,
	}
	chain, err := cosmos.NewChain(&cfg.Cosmos, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain: %w", err)
	}
	ra := &loop.RelayerAdapter{Relayer: cosmos.NewRelayer(c.Logger, chain), RelayerExt: chain}

	c.SubService(ra)

	return ra, nil
}
