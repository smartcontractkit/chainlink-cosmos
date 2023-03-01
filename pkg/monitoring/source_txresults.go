package monitoring

import (
	"context"
	"fmt"
	"sync"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/monitoring/fcdclient"
)

// NewTxResultsSourceFactory builds sources of TxResults objects expected by the relay monitoring.
func NewTxResultsSourceFactory(
	client fcdclient.Client,
) relayMonitoring.SourceFactory {
	return &txResultsSourceFactory{client}
}

type txResultsSourceFactory struct {
	client fcdclient.Client
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
		0,
		sync.Mutex{},
	}, nil
}

func (t *txResultsSourceFactory) GetType() string {
	return "txresults"
}

type txResultsSource struct {
	cosmosConfig     CosmosConfig
	cosmosFeedConfig CosmosFeedConfig
	client           fcdclient.Client

	latestTxID   uint64
	latestTxIDMu sync.Mutex
}

func (t *txResultsSource) Fetch(ctx context.Context) (interface{}, error) {
	// Query the FCD endpoint.
	response, err := t.client.GetTxList(ctx, fcdclient.GetTxListParams{
		Account: t.cosmosFeedConfig.ContractAddress,
		Limit:   10,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to fetch transactions from cosmos FCD: %w", err)
	}
	// Filter recent transactions
	// TODO (dru) keep latest processed tx in the state.
	recentTxs := []fcdclient.Tx{}
	func() {
		t.latestTxIDMu.Lock()
		defer t.latestTxIDMu.Unlock()
		maxTxID := t.latestTxID
		for _, tx := range response.Txs {
			if tx.ID > t.latestTxID {
				recentTxs = append(recentTxs, tx)
			}
			if tx.ID > maxTxID {
				maxTxID = tx.ID
			}
		}
		t.latestTxID = maxTxID
	}()
	// Count failed and succeeded recent transactions
	output := relayMonitoring.TxResults{}
	for _, tx := range recentTxs {
		if isFailedTransaction(tx) {
			output.NumFailed++
		} else {
			output.NumSucceeded++
		}
	}
	return output, nil
}

// Helpers

func isFailedTransaction(tx fcdclient.Tx) bool {
	// See https://docs.cosmos.network/master/building-modules/errors.html
	return tx.Code != 0
}
