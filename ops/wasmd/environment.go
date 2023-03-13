package wasmd

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/smartcontractkit/chainlink-cosmos/ops/utils"
	"github.com/smartcontractkit/chainlink-env/client"
	"github.com/smartcontractkit/chainlink-env/environment"
)

type Props struct {
	NetworkName string   `envconfig:"network_name"`
	HttpURLs    []string `envconfig:"http_url"`
	WsURLs      []string `envconfig:"ws_url"`
	Values      map[string]interface{}
}

type HelmProps struct {
	Name    string
	Path    string
	Version string
	Values  *map[string]interface{}
}

type Chart struct {
	HelmProps *HelmProps
	Props     *Props
}

func (m Chart) IsDeploymentNeeded() bool {
	return true
}

func (m Chart) GetProps() interface{} {
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

func (m Chart) GetValues() *map[string]interface{} {
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
	e.URLs[m.Props.NetworkName] = []string{netLocal, netLocalWS}
	if e.Cfg.InsideK8s {
		e.URLs[m.Props.NetworkName] = []string{netInternal, netInternalWS}
	}
	log.Info().Str("Name", m.Props.NetworkName).Str("URLs", netLocal).Msg("Cosmos network")
	return nil
}

func defaultProps() *Props {
	return &Props{
		NetworkName: "wasmd",
		Values: map[string]interface{}{
			"replicas": "1",
			"wasmd": map[string]interface{}{
				"image": map[string]interface{}{
					"image":   "cosmwasm/wasmd",
					"version": "v0.30.0",
				},
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"cpu":    "1000m",
						"memory": "1024Mi",
					},
					"limits": map[string]interface{}{
						"cpu":    "1000m",
						"memory": "1024Mi",
					},
				},
			},
		},
	}
}

func New(props *Props, path string) environment.ConnectedChart {
	return NewVersioned("", props, path)
}

// NewVersioned enables choosing a specific helm chart version
func NewVersioned(helmVersion string, props *Props, path string) environment.ConnectedChart {
	if props == nil {
		props = defaultProps()
	}
	if path == "" {
		path = "chainlink-qa/wasmd"
	}
	return Chart{
		HelmProps: &HelmProps{
			Name:    "wasmd",
			Path:    fmt.Sprintf("%s/charts/wasmd", utils.OpsRoot),
			Values:  &props.Values,
			Version: helmVersion,
		},
		Props: props,
	}
}
