package wasmd

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/smartcontractkit/chainlink-cosmos/ops/utils"
	"github.com/smartcontractkit/chainlink-env/client"
	"github.com/smartcontractkit/chainlink-env/config"
	"github.com/smartcontractkit/chainlink-env/environment"
)

const (
	AppName = "chainlink"
)

type Props struct{}

type HelmProps struct {
	Name    string
	Path    string
	Version string
	Values  *map[string]any
}

type Chart struct {
	HelmProps *HelmProps
	Props     *Props
}

func (m Chart) IsDeploymentNeeded() bool {
	return true
}

func (m Chart) GetProps() any {
	return m.Props
}

func (m Chart) GetName() string {
	return m.HelmProps.Name
}

func (m Chart) GetPath() string {
	return m.HelmProps.Path
}

func (m Chart) GetVersion() string {
	return m.HelmProps.Version
}

func (m Chart) GetValues() *map[string]any {
	return m.HelmProps.Values
}

func (m Chart) ExportData(e *environment.Environment) error {
	netLocal, err := e.Fwd.FindPort("wasmd:0", "wasmd", "tendermint-rpc").As(client.LocalConnection, client.HTTP)
	if err != nil {
		return err
	}
	netLocalWS, err := e.Fwd.FindPort("wasmd:0", "wasmd", "web-grpc").As(client.LocalConnection, client.WS)
	if err != nil {
		return err
	}
	netInternal, err := e.Fwd.FindPort("wasmd:0", "wasmd", "tendermint-rpc").As(client.RemoteConnection, client.HTTP)
	if err != nil {
		return err
	}
	netInternalWS, err := e.Fwd.FindPort("wasmd:0", "wasmd", "web-grpc").As(client.RemoteConnection, client.WS)
	if err != nil {
		return err
	}
	e.URLs[AppName] = []string{netLocal, netLocalWS}
	if e.Cfg.InsideK8s {
		e.URLs[AppName] = []string{netInternal, netInternalWS}
	}
	log.Info().Str("Name", AppName).Str("URLs", netLocal).Msg("Cosmos network")
	return nil
}

func defaultProps() map[string]any {
	return map[string]any{
		"replicas": "1",
		"wasmd": map[string]any{
			"image": map[string]any{
				"image":   "cosmwasm/wasmd",
				"version": "v0.30.0", // TODO: may want to ty this to an env var
			},
			"resources": map[string]any{
				"requests": map[string]any{
					"cpu":    "1000m",
					"memory": "1024Mi",
				},
				"limits": map[string]any{
					"cpu":    "1000m",
					"memory": "1024Mi",
				},
			},
		},
	}
}

func New(props *Props) environment.ConnectedChart {
	dp := defaultProps()
	if props != nil {
		config.MustMerge(&dp, props)
	}
	return Chart{
		HelmProps: &HelmProps{
			Name:    "wasmd",
			Path:    fmt.Sprintf("%s/charts/devnet", utils.OpsRoot),
			Version: "",
			Values:  &dp,
		},
		Props: props,
	}
}
