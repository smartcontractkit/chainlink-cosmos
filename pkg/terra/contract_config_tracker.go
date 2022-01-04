package terra

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

// Contract handles the OCR2 subscription needs but does not track state (state is tracked in actual OCR2 implementation)
type Contract struct {
	JobID           string
	ContractAddress sdk.AccAddress
	terra           *Client
	Transmitter     TransmissionSigner
	log             Logger
}

func NewContractTracker(spec OCR2Spec, jobID string, client *Client, lggr Logger) (*Contract, error) {
	addr, err := sdk.AccAddressFromBech32(spec.ContractID)
	if err != nil {
		return nil, errors.Wrap(err, "Error while decoding AccAddressFromBech32")
	}

	contract := Contract{
		JobID:           jobID,
		ContractAddress: addr,
		terra:           client,
		Transmitter:     spec.TransmissionSigner,
		log:             lggr,
	}

	return &contract, nil
}

// Unused, libocr will use polling
func (ct *Contract) Notify() <-chan struct{} {
	return nil
}

// LatestConfigDetails returns data by reading the contract state and is called when Notify is triggered or the config poll timer is triggered
func (ct *Contract) LatestConfigDetails(ctx context.Context) (changedInBlock uint64, configDigest types.ConfigDigest, err error) {
	queryParams := NewAbciQueryParams(ct.ContractAddress.String(), []byte(`"latest_config_details"`))
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
		return
	}
	var config ConfigDetails
	if err = json.Unmarshal(resp.Value, &config); err != nil {
		return
	}
	changedInBlock = config.BlockNumber
	configDigest = config.ConfigDigest
	return
}

// LatestConfig returns data by searching emitted events and is called in the same scenario as LatestConfigDetails
func (ct *Contract) LatestConfig(ctx context.Context, changedInBlock uint64) (types.ContractConfig, error) {
	queryStr := fmt.Sprintf("tx.height=%d AND wasm-set_config.contract_address='%s'", changedInBlock, ct.ContractAddress)
	res, err := ct.terra.clientCtx.Client.TxSearch(ctx, queryStr, false, nil, nil, "desc")
	if err != nil {
		return types.ContractConfig{}, err
	}
	if len(res.Txs) == 0 {
		return types.ContractConfig{}, fmt.Errorf("No transactions found for block %d", changedInBlock)
	}
	// fetch event and process (use first tx and \first log set)
	if len(res.Txs[0].TxResult.Events) == 0 {
		return types.ContractConfig{}, fmt.Errorf("No events found for tx %s", res.Txs[0].Hash)
	}

	for _, event := range res.Txs[0].TxResult.Events {
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
					if err := HexToByteArray(value, &output.OnchainConfig); err != nil {
						return types.ContractConfig{}, err
					}
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
	return types.ContractConfig{}, fmt.Errorf("No set_config event found for tx %s", res.Txs[0].Hash)
}

// LatestBlockHeight returns the height of the most recent block in the chain.
func (ct *Contract) LatestBlockHeight(ctx context.Context) (blockHeight uint64, err error) {
	b, err := ct.terra.clientCtx.Client.Block(ctx, nil)
	if err != nil {
		return 0, err
	}
	return uint64(b.Block.Height), nil
}
