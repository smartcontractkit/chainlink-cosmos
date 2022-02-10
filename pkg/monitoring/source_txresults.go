package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"github.com/smartcontractkit/chainlink/core/logger"
)

// NewTxResultsSourceFactory builds sources of TxResults objects expected by the relay monitoring.
func NewTxResultsSourceFactory(log logger.Logger) relayMonitoring.SourceFactory {
	return &txResultsSourceFactory{log, &http.Client{}}
}

type txResultsSourceFactory struct {
	log        logger.Logger
	httpClient *http.Client
}

func (t *txResultsSourceFactory) NewSource(
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
	return &txResultsSource{
		t.log,
		terraConfig,
		terraFeedConfig,
		t.httpClient,
		0,
		sync.Mutex{},
	}, nil
}

type txResultsSource struct {
	log             logger.Logger
	terraConfig     TerraConfig
	terraFeedConfig TerraFeedConfig
	httpClient      *http.Client

	latestTxID   uint64
	latestTxIDMu sync.Mutex
}

type fcdTxsResponse struct {
	Txs []fcdTx `json:"txs"`
}

type fcdTx struct {
	ID uint64 `json:"id"`
	// Error code if present
	Code      int    `json:"code,omitempty"`
	CodeSpace string `json:"codespace,omitempty"`
}

func (t *txResultsSource) Fetch(ctx context.Context) (interface{}, error) {
	// Query the FCD endpoint.
	query := url.Values{}
	query.Set("account", t.terraFeedConfig.ContractAddressBech32)
	query.Set("limit", "100")
	query.Set("offset", "0")
	getTxsURL := fmt.Sprintf("%sv1/txs?%s", t.terraConfig.FCDURL, query.Encode())
	readTxsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, getTxsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to build a request to the terra FCD: %w", err)
	}
	res, err := t.httpClient.Do(readTxsReq)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch transactions from terra FCD: %w", err)
	}
	defer res.Body.Close()
	// Decode the response
	txsResponse := fcdTxsResponse{}
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&txsResponse); err != nil {
		return nil, fmt.Errorf("unable to decode transactions from response: %w", err)
	}
	// Filter recent transactions
	recentTxs := []fcdTx{}
	func() {
		t.latestTxIDMu.Lock()
		defer t.latestTxIDMu.Unlock()
		maxTxID := t.latestTxID
		for _, tx := range txsResponse.Txs {
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
		// See https://github.com/terra-money/core/blob/main/x/wasm/types/errors.go
		if tx.Code > 2 && tx.CodeSpace == "wasm" {
			output.NumFailed++
		} else {
			output.NumSucceeded++
		}
	}
	return output, nil
}
