package monitoring

import (
	"net/http"
	"sync"
	"time"

	jsonrpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"go.uber.org/ratelimit"

	cosmosclient "github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/client"
)

// ChainReader is a subset of the pkg/cosmos/client.Reader interface.
type ChainReader interface {
	Account(address sdk.AccAddress) (uint64, uint64, error)
	TxsEvents(events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error)
	ContractState(contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error)
}

// NewChainReader produces a ChainReader that issues requests to the Cosmos RPC
// in sequence, even if it's called by multiple sources in parallel.
// That's because the Cosmos endpoint is aggresively rate limitting the monitor.
func NewThrottledChainReader(cosmosConfig CosmosConfig, coreLog logger.Logger) (ChainReader, error) {
	httpClient, err := jsonrpcclient.DefaultHTTPClient(cosmosConfig.TendermintURL)
	if err != nil {
		return nil, err
	}
	httpTransport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		return nil, errors.New("invalid HTTP transport")
	}
	// Use a new connection per request.
	// ref:
	//   https://github.com/golang/go/blob/905b58b5377e8f542590a46a3c90146ab45a6c96/src/net/http/transport.go#L184
	//   https://github.com/golang/go/blob/905b58b5377e8f542590a46a3c90146ab45a6c96/src/net/http/transport.go#L930
	httpTransport.DisableKeepAlives = true
	httpTransport.MaxIdleConnsPerHost = -1

	cosmosClient, err := cosmosclient.NewClientWithHttpClient(cosmosConfig.ChainID, cosmosConfig.TendermintURL, httpClient, coreLog)
	if err != nil {
		return nil, err
	}
	return &throttledChainReader{
		cosmosClient,
		cosmosConfig,
		coreLog,
		sync.Mutex{},
		ratelimit.New(
			cosmosConfig.TendermintReqsPerSec,
			ratelimit.Per(1*time.Second),
			ratelimit.WithoutSlack, // don't accumulate previously "unspent" requests for future bursts
		),
	}, nil
}

type throttledChainReader struct {
	cosmosClient *cosmosclient.Client
	cosmosConfig CosmosConfig
	coreLog      logger.Logger

	globalSequencer sync.Mutex
	rateLimiter     ratelimit.Limiter
}

func (c *throttledChainReader) TxsEvents(events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error) {
	c.globalSequencer.Lock()
	defer c.globalSequencer.Unlock()
	_ = c.rateLimiter.Take()
	return c.cosmosClient.TxsEvents(events, paginationParams)
}

func (c *throttledChainReader) ContractState(contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error) {
	c.globalSequencer.Lock()
	defer c.globalSequencer.Unlock()
	_ = c.rateLimiter.Take()
	return c.cosmosClient.ContractState(contractAddress, queryMsg)
}

func (c *throttledChainReader) Account(address sdk.AccAddress) (uint64, uint64, error) {
	c.globalSequencer.Lock()
	defer c.globalSequencer.Unlock()
	_ = c.rateLimiter.Take()
	return c.cosmosClient.Account(address)
}
