package monitoring

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"go.uber.org/multierr"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/cosmwasm"
	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

// NewEnvelopeSourceFactory build a new object that reads observations and
// configurations from the Cosmos chain.
func NewEnvelopeSourceFactory(
	rpcClient ChainReader,
	log relayMonitoring.Logger,
) relayMonitoring.SourceFactory {
	return &envelopeSourceFactory{rpcClient, log}
}

type envelopeSourceFactory struct {
	rpcClient ChainReader
	log       relayMonitoring.Logger
}

func (e *envelopeSourceFactory) NewSource(
	chainConfig relayMonitoring.ChainConfig,
	feedConfig relayMonitoring.FeedConfig,
) (relayMonitoring.Source, error) {
	cosmosConfig, ok := chainConfig.(CosmosConfig)
	if !ok {
		return nil, fmt.Errorf("expected chainConfig to be of type CosmosConfig not %T", chainConfig)
	}
	cosmosFeedConfig, ok := feedConfig.(CosmosFeedConfig)
	if !ok {
		return nil, fmt.Errorf("expected feedConfig to be of type CosmosFeedConfig not %T", feedConfig)
	}
	return &envelopeSource{
		e.rpcClient,
		e.log,
		cosmosConfig,
		cosmosFeedConfig,

		sync.Mutex{},
		types.ContractConfig{}, // initial value for cached ContractConfig
		0,                      // initial value for the block height of the latest cached contract config
	}, nil
}

func (e *envelopeSourceFactory) GetType() string {
	return "envelope"
}

type envelopeSource struct {
	rpcClient        ChainReader
	log              relayMonitoring.Logger
	cosmosConfig     CosmosConfig
	cosmosFeedConfig CosmosFeedConfig

	cachedConfigMu    sync.Mutex
	cachedConfig      types.ContractConfig
	cachedConfigBlock uint64
}

func (e *envelopeSource) Fetch(ctx context.Context) (interface{}, error) {
	envelope := relayMonitoring.Envelope{}
	var envelopeErr error
	envelopeMu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(4)
	go func() {
		defer wg.Done()
		data, err := e.fetchLatestTransmission(ctx)
		envelopeMu.Lock()
		defer envelopeMu.Unlock()
		if err != nil {
			envelopeErr = multierr.Combine(envelopeErr, fmt.Errorf("failed to fetch transmission: %w", err))
			return
		}
		envelope.ConfigDigest = data.configDigest
		envelope.Epoch = data.epoch
		envelope.Round = data.round
		envelope.LatestAnswer = data.latestAnswer
		envelope.LatestTimestamp = data.latestTimestamp
		// Note: block number is read from the transmission transaction, not set_config!
		envelope.BlockNumber = data.blockNumber
		envelope.Transmitter = data.transmitter
		envelope.JuelsPerFeeCoin = data.juelsPerFeeCoin
		envelope.AggregatorRoundID = data.aggregatorRoundID
	}()
	go func() {
		defer wg.Done()
		contractConfig, err := e.fetchLatestConfig(ctx)
		envelopeMu.Lock()
		defer envelopeMu.Unlock()
		if err != nil {
			envelopeErr = multierr.Combine(envelopeErr, fmt.Errorf("failed to fetch config: %w", err))
			return
		}
		envelope.ContractConfig = contractConfig
	}()
	go func() {
		defer wg.Done()
		balance, err := e.fetchLinkBalance(ctx)
		envelopeMu.Lock()
		defer envelopeMu.Unlock()
		if err != nil {
			envelopeErr = multierr.Combine(envelopeErr, fmt.Errorf("failed to fetch link balance: %w", err))
			return
		}
		envelope.LinkBalance = balance
	}()
	go func() {
		defer wg.Done()
		amount, err := e.fetchLinkAvailableForPayment(ctx)
		envelopeMu.Lock()
		defer envelopeMu.Unlock()
		if err != nil {
			envelopeErr = multierr.Combine(envelopeErr, fmt.Errorf("failed to fetch link balance: %w", err))
			return
		}
		envelope.LinkAvailableForPayment = amount
	}()

	wg.Wait()
	return envelope, envelopeErr
}

type transmissionData struct {
	configDigest      types.ConfigDigest
	epoch             uint32
	round             uint8
	latestAnswer      *big.Int
	latestTimestamp   time.Time
	blockNumber       uint64
	transmitter       types.Account
	aggregatorRoundID uint32
	juelsPerFeeCoin   *big.Int
}

func (e *envelopeSource) fetchLatestTransmission(ctx context.Context) (transmissionData, error) {
	res, err := e.rpcClient.TxsEvents(
		[]string{
			fmt.Sprintf("wasm._contract_address='%s'", e.cosmosFeedConfig.ContractAddressBech32),
		},
		&query.PageRequest{
			Limit: 10, // there should be a new transmission in the last 10 blocks
		})
	if err != nil {
		return transmissionData{}, fmt.Errorf("failed to fetch latest 'new_transmission' event: %w", err)
	}
	data := transmissionData{}
	blockHeight, err := e.extractDataFromTxResponse("wasm-new_transmission", e.cosmosFeedConfig.ContractAddressBech32, res, map[string]func(string) error{
		"config_digest": func(value string) error {
			return cosmwasm.HexToConfigDigest(value, &data.configDigest)
		},
		"epoch": func(value string) error {
			rawEpoch, parseErr := strconv.ParseUint(value, 10, 32)
			data.epoch = uint32(rawEpoch)
			return parseErr
		},
		"round": func(value string) error {
			rawRound, parseErr := strconv.ParseUint(value, 10, 8)
			data.round = uint8(rawRound)
			return parseErr
		},
		"answer": func(value string) error {
			var success bool
			data.latestAnswer, success = new(big.Int).SetString(value, 10)
			if !success {
				return fmt.Errorf("failed to read latest answer from value '%s'", value)
			}
			return nil
		},
		"observations_timestamp": func(value string) error {
			rawTimestamp, parseErr := strconv.ParseInt(value, 10, 64)
			data.latestTimestamp = time.Unix(rawTimestamp, 0)
			return parseErr
		},
		"transmitter": func(value string) error {
			data.transmitter = types.Account(value)
			return nil
		},
		"aggregator_round_id": func(value string) error {
			raw, pasrseErr := strconv.ParseUint(value, 10, 32)
			data.aggregatorRoundID = uint32(raw)
			return pasrseErr
		},
		"juels_per_fee_coin": func(value string) error {
			var success bool
			data.juelsPerFeeCoin, success = new(big.Int).SetString(value, 10)
			if !success {
				return fmt.Errorf("failed to parse juel per fee coin from '%s'", value)
			}
			return nil
		},
	})
	if err != nil {
		return data, fmt.Errorf("failed to extract transmission from logs: %w", err)
	}
	data.blockNumber = uint64(blockHeight)
	return data, nil
}

func (e *envelopeSource) fetchLatestConfig(ctx context.Context) (types.ContractConfig, error) {
	var cachedConfig types.ContractConfig
	var cachedConfigBlock uint64
	go func() {
		e.cachedConfigMu.Lock()
		defer e.cachedConfigMu.Unlock()
		cachedConfig = e.cachedConfig
		cachedConfigBlock = e.cachedConfigBlock
	}()
	latestConfigBlock, err := e.fetchLatestConfigBlock(ctx)
	if err != nil {
		return types.ContractConfig{}, err
	}
	if cachedConfigBlock != 0 && latestConfigBlock == cachedConfigBlock {
		return cachedConfig, nil
	}
	latestConfig, err := e.fetchLatestConfigFromLogs(ctx, int64(latestConfigBlock))
	if err != nil {
		return types.ContractConfig{}, err
	}
	// Cache the config and block height
	e.cachedConfigMu.Lock()
	defer e.cachedConfigMu.Unlock()
	e.cachedConfig = latestConfig
	e.cachedConfigBlock = latestConfigBlock
	return latestConfig, nil
}

func (e *envelopeSource) fetchLatestConfigBlock(ctx context.Context) (uint64, error) {
	resp, err := e.rpcClient.ContractState(
		e.cosmosFeedConfig.ContractAddress,
		[]byte(`{"latest_config_details":{}}`),
	)
	var details cosmwasm.ConfigDetails
	if err != nil {
		return 0, fmt.Errorf("failed to fetch config details: %w", err)
	}
	if err = json.Unmarshal(resp, &details); err != nil {
		return 0, fmt.Errorf("failed to unmarshal config details: %w", err)
	}
	return details.BlockNumber, nil
}

func (e *envelopeSource) fetchLatestConfigFromLogs(ctx context.Context, blockHeight int64) (types.ContractConfig, error) {
	res, err := e.rpcClient.TxsEvents(
		[]string{
			fmt.Sprintf("tx.height=%d", blockHeight),
			fmt.Sprintf("wasm._contract_address='%s'", e.cosmosFeedConfig.ContractAddressBech32),
		},
		&query.PageRequest{
			Limit: 10,
		})
	if err != nil {
		return types.ContractConfig{}, fmt.Errorf("failed to fetch block at height %d: %w", blockHeight, err)
	}
	output := types.ContractConfig{}
	_, err = e.extractDataFromTxResponse("wasm-set_config", e.cosmosFeedConfig.ContractAddressBech32, res, map[string]func(string) error{
		"latest_config_digest": func(value string) error {
			// parse byte array encoded as hex string
			return cosmwasm.HexToConfigDigest(value, &output.ConfigDigest)
		},
		"config_count": func(value string) error {
			i, parseErr := strconv.ParseInt(value, 10, 64)
			output.ConfigCount = uint64(i)
			return parseErr
		},
		"signers": func(value string) error {
			// this assumes the value will be a hex encoded string which each signer
			// 32 bytes and each signer will be a separate parameter
			var v []byte
			convertErr := cosmwasm.HexToByteArray(value, &v)
			output.Signers = append(output.Signers, v)
			return convertErr
		},
		"transmitters": func(value string) error {
			// this assumes the return value be a string for each transmitter and each transmitter will be separate
			output.Transmitters = append(output.Transmitters, types.Account(value))
			return nil
		},
		"f": func(value string) error {
			i, parseErr := strconv.ParseInt(value, 10, 8)
			output.F = uint8(i)
			return parseErr
		},
		"onchain_config": func(value string) error {
			// parse byte array encoded as base64
			config, decodeErr := base64.StdEncoding.DecodeString(value)
			output.OnchainConfig = config
			return decodeErr
		},
		"offchain_config_version": func(value string) error {
			i, parseErr := strconv.ParseInt(value, 10, 64)
			output.OffchainConfigVersion = uint64(i)
			return parseErr
		},
		"offchain_config": func(value string) error {
			// parse byte array encoded as hex string
			config, convertErr := base64.StdEncoding.DecodeString(value)
			output.OffchainConfig = config
			return convertErr
		},
	})
	if err != nil {
		return types.ContractConfig{}, fmt.Errorf("failed to extract config from logs: %w", err)
	}
	return output, nil
}

type linkBalanceResponse struct {
	Balance string `json:"balance"`
}

func (e *envelopeSource) fetchLinkBalance(ctx context.Context) (*big.Int, error) {
	query := fmt.Sprintf(`{"balance":{"address":"%s"}}`, e.cosmosFeedConfig.ContractAddressBech32)
	res, err := e.rpcClient.ContractState(
		e.cosmosConfig.LinkTokenAddress,
		[]byte(query),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch balance: %w", err)
	}
	balanceRes := linkBalanceResponse{}
	if err = json.Unmarshal(res, &balanceRes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal balance response: %w", err)
	}
	balance, success := new(big.Int).SetString(balanceRes.Balance, 10)
	if !success {
		return nil, fmt.Errorf("failed to parse link balance from '%s'", balanceRes.Balance)
	}
	return balance, nil
}

type linkAvailableForPaymentRes struct {
	Amount string `json:"amount,omitempty"`
}

func (e *envelopeSource) fetchLinkAvailableForPayment(ctx context.Context) (*big.Int, error) {
	res, err := e.rpcClient.ContractState(
		e.cosmosFeedConfig.ContractAddress,
		[]byte(`{"link_available_for_payment":{}}`),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read link_available_for_payment from the contract: %w", err)
	}
	linkAvailableForPayment := linkAvailableForPaymentRes{}
	if err := json.Unmarshal(res, &linkAvailableForPayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal link available data from the response '%s': %w", string(res), err)
	}
	amount, success := new(big.Int).SetString(linkAvailableForPayment.Amount, 10)
	if !success {
		return nil, fmt.Errorf("failed to parse amount of link available for payment from string '%s' into a big.Int", linkAvailableForPayment.Amount)
	}
	return amount, nil
}

// Helpers

func (e *envelopeSource) extractDataFromTxResponse(
	eventType string,
	contractAddressBech32 string,
	res *tx.GetTxsEventResponse,
	extractors map[string]func(string) error,
) (int64, error) {
	// Extract matching events
	blockHeight, event, err := findMatchingEvent(res, eventType, contractAddressBech32)
	if err != nil {
		return blockHeight, err
	}
	hasKnownAttribute := false
	// Apply extractors.
	// Note! If multiple attributes with the same key are present, the corresponding
	// extractor fn will be called for each of them.
	for _, attribute := range event.Attributes {
		key, value := attribute.Key, attribute.Value
		extractor, found := extractors[key]
		if !found {
			continue
		}
		if err := extractor(value); err != nil {
			return blockHeight, fmt.Errorf("failed to extract '%s' from raw value '%s': %w", key, value, err)
		}
		hasKnownAttribute = true
	}
	if !hasKnownAttribute {
		return blockHeight, fmt.Errorf("no known attributes found in event with type='%s' and contract_address='%s'", eventType, contractAddressBech32)
	}
	return blockHeight, nil
}

func findMatchingEvent(res *tx.GetTxsEventResponse, eventType, contractAddressBech32 string) (int64, cosmostypes.StringEvent, error) {
	// Events are already returned in reverse chronological order by TxsEvents().
	for _, txResponse := range res.TxResponses {
		if len(txResponse.Logs) == 0 {
			continue
		}
		for _, event := range txResponse.Logs[0].Events {
			if event.Type != eventType {
				continue
			}
			for _, attribute := range event.Attributes {
				if attribute.Key == "_contract_address" && attribute.Value == contractAddressBech32 {
					return txResponse.Height, event, nil
				}
			}
		}
	}
	return 0, cosmostypes.StringEvent{}, fmt.Errorf("no event found with type='%s' and contract_address='%s'", eventType, contractAddressBech32)
}
