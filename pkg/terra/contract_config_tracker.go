package terra

import (
	"context"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
)

var _ types.ContractConfigTracker = (*ContractTracker)(nil)

type ContractTracker struct {
	chainReader client.Reader
	contract    *ContractCache
}

func NewContractTracker(chainReader client.Reader, state *ContractCache) *ContractTracker {
	contract := ContractTracker{
		chainReader: chainReader,
		contract:    state,
	}
	return &contract
}

// Unused, libocr will use polling
func (ct *ContractTracker) Notify() <-chan struct{} {
	return nil
}

// LatestConfigDetails returns data by reading the address state and is called when Notify is triggered or the config poll timer is triggered
func (ct *ContractTracker) LatestConfigDetails(ctx context.Context) (changedInBlock uint64, configDigest types.ConfigDigest, err error) {
	return ct.contract.latestConfigDetails()

}

// LatestConfig returns data by searching emitted events and is called in the same scenario as LatestConfigDetails
func (ct *ContractTracker) LatestConfig(_ context.Context, changedInBlock uint64) (contractConfig types.ContractConfig, err error) {
	return ct.contract.latestConfig(changedInBlock)
}

// LatestBlockHeight returns the height of the most recent block in the chain.
func (ct *ContractTracker) LatestBlockHeight(ctx context.Context) (blockHeight uint64, err error) {
	b, err := ct.chainReader.LatestBlock()
	if err != nil {
		return 0, err
	}
	return uint64(b.Block.Header.Height), nil
}
