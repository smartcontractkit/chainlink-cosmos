package terra

import (
	"context"
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

func (or *OCR2Reader) fetchLatestConfigDetails(ctx context.Context) (changedInBlock uint64, configDigest types.ConfigDigest, err error) {
	resp, err := or.chainReader.ContractStore(
		or.address,
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

func (or *OCR2Reader) fetchLatestConfig(ctx context.Context, changedInBlock uint64) (types.ContractConfig, error) {
	query := []string{fmt.Sprintf("tx.height=%d", changedInBlock), fmt.Sprintf("wasm-set_config.contract_address='%s'", or.address)}
	res, err := or.chainReader.TxsEvents(query)
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
			output := types.ContractConfig{}
			// TODO: is there a better way to parse an array of structs to an struct
			// https://github.com/smartcontractkit/chainlink-terra/issues/21
			for _, attr := range event.Attributes {
				key, value := string(attr.Key), string(attr.Value)
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
					// parse byte array encoded as hex string
					var config33 []byte
					if err := HexToByteArray(value, &config33); err != nil {
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
					// parse byte array encoded as hex string
					if err := HexToByteArray(value, &output.OffchainConfig); err != nil {
						return types.ContractConfig{}, err
					}
				}
			}
			return output, nil
		}
	}
	return types.ContractConfig{}, fmt.Errorf("No set_config event found for tx %s", res.TxResponses[0].TxHash)
}

// latestTransmissionDetails fetches the latest transmission details from address state
func (or *OCR2Reader) fetchLatestTransmissionDetails(ctx context.Context) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	latestAnswer *big.Int,
	latestTimestamp time.Time,
	err error,
) {
	resp, err := or.chainReader.ContractStore(or.address, []byte(`"latest_transmission_details"`))
	if err != nil {
		// TODO: Verify if this is still necessary
		// https://github.com/smartcontractkit/chainlink-terra/issues/23
		// Handle the 500 error that occurs when there has not been a submission
		// "rpc error: code = Unknown desc = ocr2::state::Transmission not found: address query failed"
		if strings.Contains(fmt.Sprint(err), "ocr2::state::Transmission not found") {
			or.lggr.Infof("No transmissions found when fetching `latest_transmission_details` attempting with `latest_config_digest_and_epoch`")
			digest, epoch, err2 := or.latestConfigDigestAndEpoch(ctx)

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

// fetchLatestRoundRequested fetches the latest round requested by filtering event logs
//func (cc *OCR2Reader) fetchLatestRoundRequested(ctx context.Context, lookback time.Duration) (
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
//	res, err := cc.chainReader.TxsEvents([]string{fmt.Sprintf("tx.height>=%d", blockNum+1), fmt.Sprintf("wasm-new_round.contract_address='%s'", cc.address.String())})
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

// latestConfigDigestAndEpoch fetches the latest details from address state
func (or *OCR2Reader) latestConfigDigestAndEpoch(ctx context.Context) (
	configDigest types.ConfigDigest,
	epoch uint32,
	err error,
) {
	resp, err := or.chainReader.ContractStore(
		or.address, []byte(`"latest_config_digest_and_epoch"`),
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
