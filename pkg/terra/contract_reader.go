package terra

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
)

type OCR2Reader struct {
	address     cosmosSDK.AccAddress
	chainReader client.Reader
	lggr        Logger
}

func NewOCR2Reader(addess cosmosSDK.AccAddress, chainReader client.Reader, lggr Logger) *OCR2Reader {
	return &OCR2Reader{
		address:     addess,
		chainReader: chainReader,
		lggr:        lggr,
	}
}

func (r *OCR2Reader) LatestConfigDetails(ctx context.Context) (changedInBlock uint64, configDigest types.ConfigDigest, err error) {
	resp, err := r.chainReader.ContractStore(
		r.address,
		[]byte(`"latest_config_details"`),
	)
	if err != nil {
		return
	}
	var config ConfigDetails
	if err = json.Unmarshal(resp, &config); err != nil {
		return
	}
	changedInBlock = config.BlockNumber
	configDigest = config.ConfigDigest
	return
}

func (r *OCR2Reader) LatestConfig(ctx context.Context, changedInBlock uint64) (types.ContractConfig, error) {
	query := []string{fmt.Sprintf("tx.height=%d", changedInBlock), fmt.Sprintf("wasm-set_config.contract_address='%s'", r.address)}
	res, err := r.chainReader.TxsEvents(query, nil)
	if err != nil {
		return types.ContractConfig{}, err
	}
	if len(res.TxResponses) == 0 {
		return types.ContractConfig{}, fmt.Errorf("No transactions found for block %d, query %v", changedInBlock, query)
	}
	// fetch event and process (use first tx and \first log set)
	if len(res.TxResponses[0].Logs) == 0 {
		return types.ContractConfig{}, fmt.Errorf("No logs found for tx %s, query %v", res.TxResponses[0].TxHash, query)
	}
	if len(res.TxResponses[0].Logs[0].Events) == 0 {
		return types.ContractConfig{}, fmt.Errorf("No events found for tx %s, query %v", res.TxResponses[0].TxHash, query)
	}

	for _, event := range res.TxResponses[0].Logs[0].Events {
		if event.Type == "wasm-set_config" {
			return parseAttributes(event.Attributes)
		}
	}
	return types.ContractConfig{}, fmt.Errorf("No set_config event found for tx %s", res.TxResponses[0].TxHash)
}

func parseAttributes(attrs []cosmosSDK.Attribute) (output types.ContractConfig, err error) {
	for _, attr := range attrs {
		key, value := attr.Key, attr.Value
		switch key {
		case "latest_config_digest":
			// parse byte array encoded as hex string
			if err := HexToConfigDigest(value, &output.ConfigDigest); err != nil {
				return types.ContractConfig{}, err
			}
		case "config_count":
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return types.ContractConfig{}, err
			}
			output.ConfigCount = uint64(i)
		case "signers":
			// this assumes the value will be a hex encoded string which each signer 32 bytes and each signer will be a separate parameter
			var v []byte
			if err := HexToByteArray(value, &v); err != nil {
				return types.ContractConfig{}, err
			}
			output.Signers = append(output.Signers, v)
		case "transmitters":
			// this assumes the return value be a string for each transmitter and each transmitter will be separate
			output.Transmitters = append(output.Transmitters, types.Account(attr.Value))
		case "f":
			i, err := strconv.ParseInt(value, 10, 8)
			if err != nil {
				return types.ContractConfig{}, err
			}
			output.F = uint8(i)
		case "onchain_config":
			// parse byte array encoded as base64
			config33, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				return types.ContractConfig{}, err
			}
			// convert byte array to encoding expected by lib OCR
			config49, err := ContractConfigToOCRConfig(config33)
			if err != nil {
				return types.ContractConfig{}, err

			}
			output.OnchainConfig = config49
		case "offchain_config_version":
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return types.ContractConfig{}, err
			}
			output.OffchainConfigVersion = uint64(i)
		case "offchain_config":
			// parse byte array encoded as base64
			bytes, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				return types.ContractConfig{}, err
			}
			output.OffchainConfig = bytes
		}
	}
	return output, nil
}

// LatestTransmissionDetails fetches the latest transmission details from address state
func (r *OCR2Reader) LatestTransmissionDetails(ctx context.Context) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	latestAnswer *big.Int,
	latestTimestamp time.Time,
	err error,
) {
	resp, err := r.chainReader.ContractStore(r.address, []byte(`"latest_transmission_details"`))
	if err != nil {
		// Handle the 500 error that occurs when there has not been a submission
		// "rpc error: code = Unknown desc = ocr2::state::Transmission not found: contract query failed: unknown request"
		// which is thrown if this map lookup fails https://github.com/smartcontractkit/chainlink-terra/blob/main/contracts/ocr2/src/contract.rs#L759
		if strings.Contains(fmt.Sprint(err), "ocr2::state::Transmission not found") {
			r.lggr.Infof("No transmissions found when fetching `latest_transmission_details` attempting with `latest_config_digest_and_epoch`")
			digest, epoch, err2 := r.LatestConfigDigestAndEpoch(ctx)

			// In the case that there have been no transmissions, we expect the epoch to be zero.
			// We return just the contract digest here and set the rest of the
			// transmission details to their zero value.
			if err2 == nil {
				if epoch != 0 {
					r.lggr.Errorf("unexpected non-zero epoch %v and no transmissions found contract %v", epoch, r.address)
				}
				return digest, epoch, 0, big.NewInt(0), time.Unix(0, 0), nil
			} else {
				r.lggr.Errorf("error reading latest config digest and epoch err %v contract %v",  err2, r.address)
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
//func (cc *OCR2Reader) LatestRoundRequested(ctx context.Context, lookback time.Duration) (
//	configDigest types.ConfigDigest,
//	epoch uint32,
//	round uint8,
//	err error,
//) {
//	// calculate start block
//	latestBlock, blkErr := cc.chainReader.LatestBlock()
//	if blkErr != nil {
//		err = blkErr
//		return
//	}
//	blockNum := uint64(latestBlock.Block.Header.Height) - uint64(lookback/cc.cfg.BlockRate())
//	res, err := cc.chainReader.TxsEvents([]string{fmt.Sprintf("tx.height>=%d", blockNum+1), fmt.Sprintf("wasm-new_round.contract_address='%s'", cc.address.String())}, nil)
//	if err != nil {
//		return
//	}
//	if len(res.TxResponses) == 0 {
//		return
//	}
//	if len(res.TxResponses[0].Logs) == 0 {
//		err = fmt.Errorf("No logs found for tx %s", res.TxResponses[0].TxHash)
//		return
//	}
//	// First tx is the latest.
//	if len(res.TxResponses[0].Logs[0].Events) == 0 {
//		err = fmt.Errorf("No events found for tx %s", res.TxResponses[0].TxHash)
//		return
//	}
//
//	for _, event := range res.TxResponses[0].Logs[0].Events {
//		if event.Type == "wasm-new_round" {
//			// TODO: confirm event parameters
//			// https://github.com/smartcontractkit/chainlink-terra/issues/22
//			for _, attr := range event.Attributes {
//				key, value := string(attr.Key), string(attr.Value)
//				switch key {
//				case "latest_config_digest":
//					// parse byte array encoded as hex string
//					if err := HexToConfigDigest(value, &configDigest); err != nil {
//						return configDigest, epoch, round, err
//					}
//				case "epoch":
//					epochU64, err := strconv.ParseUint(value, 10, 32)
//					if err != nil {
//						return configDigest, epoch, round, err
//					}
//					epoch = uint32(epochU64)
//				case "round":
//					roundU64, err := strconv.ParseUint(value, 10, 8)
//					if err != nil {
//						return configDigest, epoch, round, err
//					}
//					round = uint8(roundU64)
//				}
//			}
//			return // exit once all parameters are processed
//		}
//	}
//	return
//}

// LatestConfigDigestAndEpoch fetches the latest details from address state
func (r *OCR2Reader) LatestConfigDigestAndEpoch(ctx context.Context) (
	configDigest types.ConfigDigest,
	epoch uint32,
	err error,
) {
	resp, err := r.chainReader.ContractStore(
		r.address, []byte(`"latest_config_digest_and_epoch"`),
	)
	if err != nil {
		return types.ConfigDigest{}, 0, err
	}

	var digest LatestConfigDigestAndEpoch
	if err := json.Unmarshal(resp, &digest); err != nil {
		return types.ConfigDigest{}, 0, err
	}

	return digest.ConfigDigest, digest.Epoch, nil
}
