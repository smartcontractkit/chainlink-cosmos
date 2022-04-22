package monitoring

import (
	"context"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	pkgClient "github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
)

// ChainReader is a subset of the pkg/terra/client.Reader interface enhanced with context support.
type ChainReader interface {
	TxsEvents(ctx context.Context, events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error)
	ContractStore(ctx context.Context, contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error)
}

// NewChainReader produces a ChainReader that issues requests to the Terra RPC
// in sequence, even if it's called by multiple sources in parallel.
// That's because the Terra endpoint is aggresively rate limitting the monitor.
func NewChainReader(client *pkgClient.Client) ChainReader {
	return &chainReader{
		client,
		sync.Mutex{},
		sync.Mutex{},
	}
}

type chainReader struct {
	client *pkgClient.Client

	txEventsSequencer      sync.Mutex
	contractStoreSequencer sync.Mutex
}

func (c *chainReader) TxsEvents(_ context.Context, events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error) {
	c.txEventsSequencer.Lock()
	defer c.txEventsSequencer.Unlock()
	return c.client.TxsEvents(events, paginationParams)
}

func (c *chainReader) ContractStore(_ context.Context, contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error) {
	c.contractStoreSequencer.Lock()
	defer c.contractStoreSequencer.Unlock()
	return c.client.ContractStore(contractAddress, queryMsg)
}
