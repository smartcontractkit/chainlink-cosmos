package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/pelletier/go-toml/v2"

	"github.com/smartcontractkit/chainlink-common/pkg/beholder"
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

func (c *pluginRelayer) NewRelayer(ctx context.Context, config string, keystore, csaKeystore core.Keystore, capRegistry core.CapabilitiesRegistry) (loop.Relayer, error) {
	_ = csaKeystore

	d := toml.NewDecoder(strings.NewReader(config))
	d.DisallowUnknownFields()

	var cfg coscfg.TOMLConfig
	if err := d.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config toml: %w:\n\t%s", err, config)
	}
	cfg.SetDefaults()
	if err := cfg.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("config is invalid: %w", err)
	}

	cfgStr, err := cfg.TOMLString()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize config: %w", err)
	}
	c.Logger.Infow("Creating relayer", "config", cfgStr)

	rawNodes := make([]map[string]string, 0, len(cfg.Nodes))
	for _, n := range cfg.Nodes {
		if n == nil || n.TendermintURL == nil {
			continue
		}
		rawNodes = append(rawNodes, map[string]string{"TendermintURL": n.TendermintURL.String()})
	}
	chainID := ""
	if cfg.ChainID != nil {
		chainID = *cfg.ChainID
	}
	emitter := loop.NewPluginRelayerConfigEmitter(
		c.Logger,
		beholder.GetClient().Config.AuthPublicKeyHex,
		chainID,
		rawNodes,
	)
	err = emitter.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start plugin relayer config emitter: %w", err)
	}
	c.SubService(emitter)

	chain, err := cosmos.NewChain(&cfg, cosmos.ChainOpts{
		Logger:   c.Logger,
		KeyStore: keystore,
		DS:       c.ds,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chain: %w", err)
	}
	r := cosmos.NewRelayer(c.Logger, chain)

	c.SubService(r)

	return r, nil
}
