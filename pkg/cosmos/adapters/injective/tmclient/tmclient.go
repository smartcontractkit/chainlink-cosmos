package tmclient

import (
	"context"
	"strings"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"

	rpcclient "github.com/tendermint/tendermint/rpc/client"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type TendermintClient interface {
	GetBlock(ctx context.Context, height int64) (*tmctypes.ResultBlock, error)
	GetLatestBlockHeight(ctx context.Context) (int64, error)
	GetTxs(ctx context.Context, block *tmctypes.ResultBlock) ([]*ctypes.ResultTx, error)
	GetValidatorSet(ctx context.Context, height int64) (*tmctypes.ResultValidators, error)
}

type tmClient struct {
	lggr      logger.Logger
	rpcClient rpcclient.Client
}

func NewRPCClient(rpcNodeAddr string, lggr logger.Logger) TendermintClient {
	rpcClient, err := rpchttp.NewWithTimeout(rpcNodeAddr, "/websocket", 10)
	if err != nil {
		lggr.Errorw("failed to init rpcClient", "err", err)
	}

	return &tmClient{
		lggr:      lggr,
		rpcClient: rpcClient,
	}
}

// GetBlock queries for a block by height. An error is returned if the query fails.
func (c *tmClient) GetBlock(ctx context.Context, height int64) (*tmctypes.ResultBlock, error) {
	return c.rpcClient.Block(ctx, &height)
}

// GetLatestBlockHeight returns the latest block height on the active chain.
func (c *tmClient) GetLatestBlockHeight(ctx context.Context) (int64, error) {
	status, err := c.rpcClient.Status(ctx)
	if err != nil {
		return -1, err
	}

	height := status.SyncInfo.LatestBlockHeight

	return height, nil
}

// GetTxs queries for all the transactions in a block height.
// It uses `Tx` RPC method to query for the transaction.
func (c *tmClient) GetTxs(ctx context.Context, block *tmctypes.ResultBlock) ([]*ctypes.ResultTx, error) {
	txs := make([]*ctypes.ResultTx, 0, len(block.Block.Txs))
	for _, tmTx := range block.Block.Txs {
		tx, err := c.rpcClient.Tx(ctx, tmTx.Hash(), true)
		if err != nil {
			if strings.HasSuffix(err.Error(), "not found") {
				c.lggr.Errorw("failed to get tx by hash", "err", err)
				continue
			}

			return nil, err
		}

		txs = append(txs, tx)
	}

	return txs, nil
}

// GetValidatorSet returns all the known Tendermint validators for a given block
// height. An error is returned if the query fails.
func (c *tmClient) GetValidatorSet(ctx context.Context, height int64) (*tmctypes.ResultValidators, error) {
	return c.rpcClient.Validators(ctx, &height, nil, nil)
}
