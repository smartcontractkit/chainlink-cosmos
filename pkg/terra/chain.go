package terra

import (
	"github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
)

type ChainSet interface {
	Service
	Chain(id string) (Chain, error)
}

type Chain interface {
	Service

	ID() string
	Config() Config
	TxManager() TxManager
	// Reader returns a new Reader. If nodeName is provided, the underlying client must use that node.
	Reader(nodeName string) (client.Reader, error)
}

type Service interface {
	Start() error
	Close() error
	Ready() error
	Healthy() error
}
