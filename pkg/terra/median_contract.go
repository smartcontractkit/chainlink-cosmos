package terra

import (
	"context"
	"encoding/json"
	"fmt"

	abci "github.com/tendermint/tendermint/abci/types"

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

// LatestTransmissionDetails fetches the latest transmission details from contract state
func (ct *Contract) LatestTransmissionDetails(
	ctx context.Context,
) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	latestAnswer *big.Int,
	latestTimestamp time.Time,
	err error,
) {
	queryParams := NewAbciQueryParams(ct.ContractAddress.String(), []byte(`"latest_transmission_details"`))
	data, err := ct.terra.codec.MarshalJSON(queryParams)
	if err != nil {
		return
	}
	resp, err := ct.terra.clientCtx.QueryABCI(abci.RequestQuery{
		Data:   data,
		Path:   "custom/wasm/contractStore",
		Height: 0,
		Prove:  false,
	})
	if err != nil {
		// TODO: Verify if this is still necessary
		// https://github.com/smartcontractkit/chainlink-terra/issues/23
		// Handle the 500 error that occurs when there has not been a submission
		// "rpc error: code = Unknown desc = ocr2::state::Transmission not found: contract query failed"
		if strings.Contains(fmt.Sprint(err), "ocr2::state::Transmission not found") {
			ct.log.Infof("No transmissions found when fetching `latest_transmission_details` attempting with `latest_config_digest_and_epoch`")
			digest, epoch, err2 := ct.LatestConfigDigestAndEpoch(ctx)

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
	if err := json.Unmarshal(resp.Value, &details); err != nil {
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
func (ct *Contract) LatestRoundRequested(ctx context.Context, lookback time.Duration) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	err error,
) {
	// calculate start block
	blockNum := ct.terra.Height - uint64(lookback.Seconds())/BlockRate
	queryStr := fmt.Sprintf("tx.height > %d AND wasm-new_round.contract_address='%s'", blockNum, ct.ContractAddress)
	res, err := ct.terra.clientCtx.Client.TxSearch(ctx, queryStr, false, nil, nil, "desc")
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
