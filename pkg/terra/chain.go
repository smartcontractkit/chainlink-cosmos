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
	MsgEnqueuer() MsgEnqueuer
	Reader() client.Reader
}

type Service interface {
	Start() error
	Close() error
	Ready() error
	Healthy() error
}
