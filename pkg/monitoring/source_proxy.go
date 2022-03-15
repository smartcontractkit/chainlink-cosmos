package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"go.uber.org/multierr"
)

// ProxyData is a subset of the data returned by the Terra feed proxy contract's "latest_round_data" method.
type ProxyData struct {
	Answer *big.Int
	// This value is not taken from the proxy contract but from the backend contract.
	// It's added here for convenience.
	LinkAvailableForPayment *big.Int
}

// NewProxySourceFactory does wisott.
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

func (p *proxySourceFactory) GetType() string {
	return "proxy"
}

type proxySource struct {
	client          ChainReader
	log             relayMonitoring.Logger
	terraConfig     TerraConfig
	terraFeedConfig TerraFeedConfig
}

func (p *proxySource) Fetch(ctx context.Context) (interface{}, error) {
	if p.terraFeedConfig.ProxyAddressBech32 == "" {
		p.log.Debugw("skipping fetch because no proxy contract is configured", "feed", p.terraFeedConfig.ContractAddressBech32)
		return nil, relayMonitoring.ErrNoUpdate
	}
	proxyData := ProxyData{}
	var proxyErr error
	proxyDataMu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		answer, err := p.fetchLatestRoundFromProxy(ctx)
		proxyDataMu.Lock()
		defer proxyDataMu.Unlock()
		if err != nil {
			proxyErr = multierr.Combine(proxyErr, err)
		} else {
			proxyData.Answer = answer
		}
	}()
	go func() {
		defer wg.Done()
		amount, err := p.fetchLinkAvailableForPayment(ctx)
		proxyDataMu.Lock()
		defer proxyDataMu.Unlock()
		if err != nil {
			proxyErr = multierr.Combine(proxyErr, err)
		} else {
			proxyData.LinkAvailableForPayment = amount
		}
	}()
	wg.Wait()
	return proxyData, proxyErr
}

// latestRoundDataRes corresponds to a subset of the Round type in the proxy contract.
type latestRoundDataRes struct {
	Answer string `json:"answer,omitempty"`
}

func (p *proxySource) fetchLatestRoundFromProxy(ctx context.Context) (*big.Int, error) {
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
	return answer, nil
}

type linkAvailableForPaymentRes struct {
	Amount string `json:"amount,omitempty"`
}

func (p *proxySource) fetchLinkAvailableForPayment(ctx context.Context) (*big.Int, error) {
	res, err := p.client.ContractStore(
		p.terraFeedConfig.ContractAddress,
		[]byte(`"link_available_for_payment"`),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read link_available_for_payment from the proxy contract: %w", err)
	}
	linkAvailableForPayment := linkAvailableForPaymentRes{}
	if err := json.Unmarshal(res, &linkAvailableForPayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal link available data from the response '%s': %w", string(res), err)
	}
	amount, success := new(big.Int).SetString(linkAvailableForPayment.Amount, 10)
	if !success {
		return nil, fmt.Errorf("failed to parse amount of link available for payment from string '%s' into a big.Int", linkAvailableForPayment.Amount)
	}
	return amount, nil
}
