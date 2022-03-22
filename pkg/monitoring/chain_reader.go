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

func (c *chainReader) TxsEvents(ctx context.Context, events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error) {
	c.txEventsSequencer.Lock()
	defer c.txEventsSequencer.Unlock()
	raw, err := withContext(ctx, func() (interface{}, error) {
		return c.client.TxsEvents(events, paginationParams)
	})
	if err != nil {
		return nil, err
	}
	return raw.(*txtypes.GetTxsEventResponse), nil
}

func (c *chainReader) ContractStore(ctx context.Context, contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error) {
	c.contractStoreSequencer.Lock()
	defer c.contractStoreSequencer.Unlock()
	raw, err := withContext(ctx, func() (interface{}, error) {
		return c.client.ContractStore(contractAddress, queryMsg)
	})
	if err != nil {
		return nil, err
	}
	return raw.([]byte), nil
}

type callResult struct {
	data interface{}
	err  error
}

// Helpers

// withContext makes a function that does not take in a context exit when the context cancels or expires.
// In reality withContext will abandon a call that continues after the context expires and simply return an error.
// This helper is needed, because the cosmos/tendermint clients don't respect context.
// Note! This method may leak goroutines.
func withContext(ctx context.Context, call func() (interface{}, error)) (interface{}, error) {
	callResults := make(chan callResult)
	go func() {
		data, err := call()
		select {
		case callResults <- callResult{data, err}:
		case <-ctx.Done():
		}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-callResults:
		return result.data, result.err
	}
}
