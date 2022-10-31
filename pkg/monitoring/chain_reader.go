package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"go.uber.org/ratelimit"

	pkgClient "github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
)

// ChainReader is a subset of the pkg/terra/client.Reader interface enhanced with context support.
type ChainReader interface {
	TxsEvents(ctx context.Context, events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error)
	ContractState(ctx context.Context, contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error)
}

// NewChainReader produces a ChainReader that issues requests to the Terra RPC
// in sequence, even if it's called by multiple sources in parallel.
// That's because the Terra endpoint is aggresively rate limitting the monitor.
func NewChainReader(terraConfig TerraConfig, coreLog logger.Logger) ChainReader {
	return &chainReader{
		terraConfig,
		coreLog,
		sync.Mutex{},
		ratelimit.New(
			terraConfig.TendermintReqsPerSec,
			ratelimit.Per(1*time.Second),
			ratelimit.WithoutSlack, // don't accumulate previously "unspent" requests for future bursts
		),
	}
}

type chainReader struct {
	terraConfig TerraConfig
	coreLog     logger.Logger

	globalSequencer sync.Mutex
	rateLimiter     ratelimit.Limiter
}

func (c *chainReader) TxsEvents(_ context.Context, events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error) {
	c.globalSequencer.Lock()
	defer c.globalSequencer.Unlock()
	client, err := pkgClient.NewClient(
		c.terraConfig.ChainID,
		c.terraConfig.TendermintURL,
		c.terraConfig.ReadTimeout,
		c.coreLog,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create a terra client: %w", err)
	}
	_ = c.rateLimiter.Take()
	return client.TxsEvents(events, paginationParams)
}

func (c *chainReader) ContractState(_ context.Context, contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error) {
	c.globalSequencer.Lock()
	defer c.globalSequencer.Unlock()
	client, err := pkgClient.NewClient(
		c.terraConfig.ChainID,
		c.terraConfig.TendermintURL,
		c.terraConfig.ReadTimeout,
		c.coreLog,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create a terra client: %w", err)
	}
	_ = c.rateLimiter.Take()
	return client.ContractState(contractAddress, queryMsg)
}
