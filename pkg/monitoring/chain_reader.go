package monitoring

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	pkgClient "github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
	"github.com/smartcontractkit/chainlink/core/logger"
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
	return &chainReader{
		terraConfig,
		coreLog,
	}
}

type chainReader struct {
	terraConfig TerraConfig
	coreLog     logger.Logger
}

func (c *chainReader) TxsEvents(events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error) {
	client, err := pkgClient.NewClient(
		c.terraConfig.ChainID,
		c.terraConfig.TendermintURL,
		c.terraConfig.ReadTimeout,
		c.coreLog,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create a terra client: %w", err)
	}
	return client.TxsEvents(events, paginationParams)
}

func (c *chainReader) ContractStore(contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error) {
	client, err := pkgClient.NewClient(
		c.terraConfig.ChainID,
		c.terraConfig.TendermintURL,
		c.terraConfig.ReadTimeout,
		c.coreLog,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create a terra client: %w", err)
	}
	return client.ContractStore(contractAddress, queryMsg)
}
