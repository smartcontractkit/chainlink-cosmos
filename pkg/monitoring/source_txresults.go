package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	errors "github.com/pkg/errors"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
)

// NewTxResultsSourceFactory builds sources of TxResults objects expected by the relay monitoring.
func NewTxResultsSourceFactory(
	client ChainReader,
) relayMonitoring.SourceFactory {
	return &txResultsSourceFactory{client}
}

type txResultsSourceFactory struct {
	client ChainReader
}

func (t *txResultsSourceFactory) NewSource(
	chainConfig relayMonitoring.ChainConfig,
	feedConfig relayMonitoring.FeedConfig,
) (relayMonitoring.Source, error) {
	cosmosConfig, ok := chainConfig.(CosmosConfig)
	if !ok {
		return nil, fmt.Errorf("expected chainConfig to be of type CosmosConfig not %T", chainConfig)
	}
	cosmosFeedConfig, ok := feedConfig.(CosmosFeedConfig)
	if !ok {
		return nil, fmt.Errorf("expected feedConfig to be of type CosmosFeedConfig not %T", feedConfig)
	}
	return &txResultsSource{
		cosmosConfig,
		cosmosFeedConfig,
		t.client,
		sync.Mutex{},
		0,
	}, nil
}

func (t *txResultsSourceFactory) GetType() string {
	return "txresults"
}

type txResultsSource struct {
	cosmosConfig     CosmosConfig
	cosmosFeedConfig CosmosFeedConfig
	client           ChainReader

	prevRoundIDMu sync.Mutex
	prevRoundID   uint32
}

func (t *txResultsSource) Fetch(ctx context.Context) (interface{}, error) {
	t.prevRoundIDMu.Lock()
	defer t.prevRoundIDMu.Unlock()

	resp, err := t.client.ContractState(t.cosmosFeedConfig.ContractAddress, []byte(`{"latest_round_data":{}}`))
	if err != nil {
		return nil, fmt.Errorf("failed to read latest round data: %w", err)
	}

	latestRoundData := struct {
		RoundID *uint32 `json:"round_id"`
	}{}
	err = json.Unmarshal(resp, &latestRoundData)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize round data: %w", err)
	}

	if latestRoundData.RoundID == nil {
		return nil, errors.New("round data missing round ID")
	}
	newRoundID := *latestRoundData.RoundID

	var numSucceeded uint32
	if t.prevRoundID != 0 {
		numSucceeded = newRoundID - t.prevRoundID
	}
	t.prevRoundID = newRoundID

	// Note that failed/rejected transactions count is always set to 0 because there is no way to count them.
	return relayMonitoring.TxResults{NumSucceeded: uint64(numSucceeded), NumFailed: 0}, nil
}
