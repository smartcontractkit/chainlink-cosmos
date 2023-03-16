package adapters

import (
	"context"

	"github.com/smartcontractkit/chainlink-relay/pkg/types"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/client"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/config"
)

type ChainSet interface {
	types.Service
	// Chain returns chain for the given id.
	Chain(ctx context.Context, id string) (Chain, error)
}

type Chain interface {
	types.Service

	ID() string
	Config() config.Config
	TxManager() TxManager
	// Reader returns a new Reader. If nodeName is provided, the underlying client must use that node.
	Reader(nodeName string) (client.Reader, error)
}
