package terra

import (
	"context"
	"math/big"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

// MedianContract interface
var _ median.MedianContract = (*MedianContract)(nil)

type MedianContract struct {
	contract *ContractCache
}

func NewMedianContract(contract *ContractCache) *MedianContract {
	return &MedianContract{contract: contract}
}

// LatestTransmissionDetails fetches the latest cached transmission details
func (ct *MedianContract) LatestTransmissionDetails(
	ctx context.Context,
) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	latestAnswer *big.Int,
	latestTimestamp time.Time,
	err error,
) {
	return ct.contract.latestTransmissionDetails()
}

// LatestRoundRequested fetches the latest round requested from the cache
func (ct *MedianContract) LatestRoundRequested(ctx context.Context, lookback time.Duration) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	err error,
) {
	return ct.contract.latestRoundRequested(lookback)
}
