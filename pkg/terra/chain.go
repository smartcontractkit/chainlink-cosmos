package terra

import (
	"github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
	"github.com/smartcontractkit/chainlink-terra/pkg/terra/config"
)

type ChainSet interface {
	Service
	Get(id string) (Chain, error)
}

type Chain interface {
	Service

	ID() string
	Config() config.ChainCfg
	MsgEnqueuer() MsgEnqueuer
	Reader() client.Reader
}

type Service interface {
	Start() error
	Close() error
	Ready() error
	Healthy() error
}
