package monitoring

import (
	"fmt"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	pkgClient "github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
	"github.com/smartcontractkit/chainlink/core/logger"
	"go.uber.org/ratelimit"
)

// ChainReader is a subset of the pkg/terra/client.Reader interface enhanced with context support.
type ChainReader interface {
	TxsEvents(events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error)
	ContractStore(contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error)
}

// NewChainReader produces a ChainReader that issues requests to the Terra RPC
// in sequence, even if it's called by multiple sources in parallel.
// That's because the Terra endpoint is aggresively rate limitting the monitor.
func NewChainReader(
	terraConfig TerraConfig,
	coreLog logger.Logger,
) ChainReader {
	readers := []ChainReader{}
	for _, tendermintURL := range terraConfig.TendermintURLs {
		readers = append(readers, &chainReader{
			terraConfig.ChainID,
			tendermintURL,
			terraConfig.ReadTimeout,
			coreLog.With("rpc-url", tendermintURL),
			sync.Mutex{},
			ratelimit.New(1, ratelimit.Per(1*time.Second)),
		})
	}
	return &multiRPCChainReader{
		terraConfig,
		coreLog,
		readers,
		-1,
		sync.Mutex{},
	}
}

type multiRPCChainReader struct {
	terraConfig TerraConfig
	coreLog     logger.Logger

	// Round-robin over multiple readers.
	readers                []ChainReader
	readersRoundRobinIndex int
	readersMu              sync.Mutex
}

func (m *multiRPCChainReader) TxsEvents(events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error) {
	reader := m.pickReader()
	return reader.TxsEvents(events, paginationParams)
}

func (m *multiRPCChainReader) ContractStore(contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error) {
	reader := m.pickReader()
	return reader.ContractStore(contractAddress, queryMsg)
}

func (m *multiRPCChainReader) pickReader() ChainReader {
	m.readersMu.Lock()
	defer m.readersMu.Unlock()
	m.readersRoundRobinIndex = (m.readersRoundRobinIndex + 1) % len(m.readers)
	return m.readers[m.readersRoundRobinIndex]
}

type chainReader struct {
	chainID        string
	tendermintURL  string
	readTimeout    time.Duration
	coreLog        logger.Logger
	callSerializer sync.Mutex
	limiter        ratelimit.Limiter
}

func (c *chainReader) TxsEvents(events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error) {
	c.callSerializer.Lock()
	defer c.callSerializer.Unlock()
	_ = c.limiter.Take()
	client, err := pkgClient.NewClient(
		c.chainID,
		c.tendermintURL,
		c.readTimeout,
		c.coreLog,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create a terra client: %w", err)
	}
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>")
	return client.TxsEvents(events, paginationParams)
}

func (c *chainReader) ContractStore(contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error) {
	c.callSerializer.Lock()
	defer c.callSerializer.Unlock()
	_ = c.limiter.Take()
	client, err := pkgClient.NewClient(
		c.chainID,
		c.tendermintURL,
		c.readTimeout,
		c.coreLog,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create a terra client: %w", err)
	}
	fmt.Println("---------------------------")
	return client.ContractStore(contractAddress, queryMsg)
}
