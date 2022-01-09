package terra

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/client"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"

	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

const (
	BlockRate = 5 // 1 block/5 seconds
)

// MedianContract interface
var _ median.MedianContract = (*MedianContract)(nil)

type LatestConfigReader interface {
	LatestConfigDigestAndEpoch(ctx context.Context) (
		configDigest types.ConfigDigest,
		epoch uint32,
		err error)
}

type MedianContract struct {
	contract sdk.AccAddress
	r        client.Reader
	lggr     Logger
	cr       LatestConfigReader
}

func NewMedianContract(contract sdk.AccAddress, r client.Reader, lggr Logger, cr LatestConfigReader) *MedianContract {
	return &MedianContract{contract: contract, r: r, lggr: lggr, cr: cr}
}

// LatestTransmissionDetails fetches the latest transmission details from contract state
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
	resp, err := ct.r.ContractStore(ct.contract.String(), []byte(`"latest_transmission_details"`))
	if err != nil {
		// TODO: Verify if this is still necessary
		// https://github.com/smartcontractkit/chainlink-terra/issues/23
		// Handle the 500 error that occurs when there has not been a submission
		// "rpc error: code = Unknown desc = ocr2::state::Transmission not found: contract query failed"
		if strings.Contains(fmt.Sprint(err), "ocr2::state::Transmission not found") {
			ct.lggr.Infof("No transmissions found when fetching `latest_transmission_details` attempting with `latest_config_digest_and_epoch`")
			digest, epoch, err2 := ct.cr.LatestConfigDigestAndEpoch(ctx)

			// return different data if no error, else continue and return previous error
			// return config digest and epoch from query, set everything else to 0
			if err2 == nil {
				return digest, epoch, 0, big.NewInt(0), time.Unix(0, 0), nil
			}
		}

		// default response if there actually is an error
		return types.ConfigDigest{}, 0, 0, big.NewInt(0), time.Now(), err
	}

	// unmarshal
	var details LatestTransmissionDetails
	if err := json.Unmarshal(resp, &details); err != nil {
		return types.ConfigDigest{}, 0, 0, big.NewInt(0), time.Now(), err
	}

	// set answer big int
	ans := new(big.Int)
	if _, success := ans.SetString(details.LatestAnswer, 10); !success {
		return types.ConfigDigest{}, 0, 0, big.NewInt(0), time.Now(), fmt.Errorf("Could not create *big.Int from %s", details.LatestAnswer)
	}

	return details.LatestConfigDigest, details.Epoch, details.Round, ans, time.Unix(details.LatestTimestamp, 0), nil
}

// LatestRoundRequested fetches the latest round requested by filtering event logs
func (ct *MedianContract) LatestRoundRequested(ctx context.Context, lookback time.Duration) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	err error,
) {
	// calculate start block
	latestBlock, blkErr := ct.r.Block(nil)
	if blkErr != nil {
		err = blkErr
		return
	}
	blockNum := uint64(latestBlock.Block.Height) - uint64(lookback.Seconds())/BlockRate
	queryStr := fmt.Sprintf("tx.height > %d AND wasm-new_round.contract_address='%s'", blockNum, ct.contract.String())
	res, err := ct.r.TxSearch(queryStr)
	if err != nil {
		return
	}
	if len(res.Txs) == 0 || res.TotalCount == 0 {
		return
	}

	// use the last one, should be the latest tx with event
	index := len(res.Txs) - 1
	if len(res.Txs[index].TxResult.Events) == 0 {
		err = fmt.Errorf("No events found for tx %s", res.Txs[index].Hash)
		return
	}

	for _, event := range res.Txs[index].TxResult.Events {
		if event.Type == "wasm-new_round" {
			// TODO: confirm event parameters
			// https://github.com/smartcontractkit/chainlink-terra/issues/22
			for _, attr := range event.Attributes {
				key, value := string(attr.Key), string(attr.Value)
				switch key {
				case "latest_config_digest":
					// parse byte array encoded as hex string
					if err := HexToConfigDigest(value, &configDigest); err != nil {
						return configDigest, epoch, round, err
					}
				case "epoch":
					epochU64, err := strconv.ParseUint(value, 10, 32)
					if err != nil {
						return configDigest, epoch, round, err
					}
					epoch = uint32(epochU64)
				case "round":
					roundU64, err := strconv.ParseUint(value, 10, 8)
					if err != nil {
						return configDigest, epoch, round, err
					}
					round = uint8(roundU64)
				}
			}
			return // exit once all parameters are processed
		}
	}
	return
}
