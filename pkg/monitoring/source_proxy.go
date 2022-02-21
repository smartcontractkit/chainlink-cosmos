package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
)

type ProxyData struct {
	Answer *big.Int
}

// NewEnvelopeSourceFactory build a new object that reads observations and
// configurations from the Terra chain.
func NewProxySourceFactory(client ChainReader, log relayMonitoring.Logger) relayMonitoring.SourceFactory {
	return &proxySourceFactory{client, log}
}

type proxySourceFactory struct {
	client ChainReader
	log    relayMonitoring.Logger
}

func (p *proxySourceFactory) NewSource(
	chainConfig relayMonitoring.ChainConfig,
	feedConfig relayMonitoring.FeedConfig,
) (relayMonitoring.Source, error) {
	terraConfig, ok := chainConfig.(TerraConfig)
	if !ok {
		return nil, fmt.Errorf("expected chainConfig to be of type TerraConfig not %T", chainConfig)
	}
	terraFeedConfig, ok := feedConfig.(TerraFeedConfig)
	if !ok {
		return nil, fmt.Errorf("expected feedConfig to be of type TerraFeedConfig not %T", feedConfig)
	}
	return &proxySource{
		p.client,
		p.log,
		terraConfig,
		terraFeedConfig,
	}, nil
}

type proxySource struct {
	client          ChainReader
	log             relayMonitoring.Logger
	terraConfig     TerraConfig
	terraFeedConfig TerraFeedConfig
}

// latestRoundDataRes corresponds to a subset of the Round type in the proxy contract.
type latestRoundDataRes struct {
	Answer string `json:"answer,omitempty"`
}

func (p *proxySource) Fetch(ctx context.Context) (interface{}, error) {
	res, err := p.client.ContractStore(
		p.terraFeedConfig.ProxyAddress,
		[]byte(`"latest_round_data"`),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read latest_round_data from the proxy contract: %w", err)
	}
	latestRoundData := latestRoundDataRes{}
	if err := json.Unmarshal(res, &latestRoundData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal round data from the response '%s': %w", string(res), err)
	}
	answer, success := new(big.Int).SetString(latestRoundData.Answer, 10)
	if !success {
		return nil, fmt.Errorf("failed to parse proxy answer '%s' into a big.Int", latestRoundData.Answer)
	}
	return ProxyData{answer}, nil
}
