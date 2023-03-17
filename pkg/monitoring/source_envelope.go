package monitoring

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"go.uber.org/multierr"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/cosmwasm"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/monitoring/lcdclient"
)

// NewEnvelopeSourceFactory build a new object that reads observations and
// configurations from the Cosmos chain.
func NewEnvelopeSourceFactory(
	rpcClient ChainReader,
	lcdClient lcdclient.Client,
	log relayMonitoring.Logger,
) relayMonitoring.SourceFactory {
	return &envelopeSourceFactory{rpcClient, lcdClient, log}
}

type envelopeSourceFactory struct {
	rpcClient ChainReader
	lcdClient lcdclient.Client
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
		e.lcdClient,
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
	lcdClient        lcdclient.Client
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
	res, err := e.lcdClient.GetTxList(ctx, lcdclient.GetTxListParams{
		Account: e.cosmosFeedConfig.ContractAddress,
		Limit:   10, // there should be a new transmission in the last 10 blocks
	})
	if err != nil {
		return transmissionData{}, fmt.Errorf("failed to fetch latest 'new_transmission' event: %w", err)
	}
	data := transmissionData{}
	err = e.extractDataFromTxResponse("wasm-new_transmission", e.cosmosFeedConfig.ContractAddressBech32, res, map[string]func(string) error{
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
	data.blockNumber, err = strconv.ParseUint(res.Txs[0].Height, 10, 64)
	if err != nil {
		return data, fmt.Errorf("failed to parse block height from lcd data '%s': %w", res.Txs[0].Height, err)
	}
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
	latestConfig, err := e.fetchLatestConfigFromLogs(ctx, latestConfigBlock)
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
		ctx,
		e.cosmosFeedConfig.ContractAddress,
		[]byte(`"latest_config_details"`),
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

func (e *envelopeSource) fetchLatestConfigFromLogs(ctx context.Context, blockHeight uint64) (types.ContractConfig, error) {
	res, err := e.lcdClient.GetBlockAtHeight(ctx, blockHeight)
	if err != nil {
		return types.ContractConfig{}, fmt.Errorf("failed to fetch block at height: %w", err)
	}
	output := types.ContractConfig{}
	err = e.extractDataFromTxResponse("wasm-set_config", e.cosmosFeedConfig.ContractAddressBech32, res, map[string]func(string) error{
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
		ctx,
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
		ctx,
		e.cosmosFeedConfig.ContractAddress,
		[]byte(`"link_available_for_payment"`),
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
	res lcdclient.Response,
	extractors map[string]func(string) error,
) error {
	// Extract matching events
	events := extractMatchingEvents(res, eventType, contractAddressBech32)
	if len(events) == 0 {
		return fmt.Errorf("no event found with type='%s' and contract_address='%s'", eventType, contractAddressBech32)
	}
	if len(events) != 1 {
		e.log.Debugw("multiple matching events found, selecting the most recent one which is the first", "type", eventType, "contract_address", contractAddressBech32)
	}
	event := events[0]
	if err := checkEventAttributes(event, extractors); err != nil {
		return fmt.Errorf("received incorrect event with type='%s' and contract_address='%s': %w", eventType, contractAddressBech32, err)
	}
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
			return fmt.Errorf("failed to extract '%s' from raw value '%s': %w", key, value, err)
		}
	}
	return nil
}

func extractMatchingEvents(res lcdclient.Response, eventType, contractAddressBech32 string) []lcdclient.Event {
	out := []lcdclient.Event{}
	// Sort txs such that the most recent tx is first
	sort.Slice(res.Txs, func(i, j int) bool {
		return res.Txs[i].ID > res.Txs[j].ID
	})
	for _, tx := range res.Txs {
		if !strings.Contains(tx.RawLog, fmt.Sprintf(`"type":"%s"`, eventType)) {
			continue
		}
		for _, event := range tx.Logs[0].Events {
			if event.Typ != eventType {
				continue
			}
			isMatchingContractAddress := false
			for _, attribute := range event.Attributes {
				if attribute.Key == "contract_address" && attribute.Value == contractAddressBech32 {
					isMatchingContractAddress = true
					break
				}
			}
			if isMatchingContractAddress {
				out = append(out, event)
			}
		}
	}
	return out
}

func checkEventAttributes(
	event lcdclient.Event,
	extractors map[string]func(string) error,
) error {
	// The event should have at least one attribute with the Key in the extractors map.
	isPresent := map[string]bool{}
	for key := range extractors {
		isPresent[key] = false
	}
	for _, attribute := range event.Attributes {
		if _, found := extractors[attribute.Key]; found {
			isPresent[attribute.Key] = true
		}
	}
	for key, found := range isPresent {
		if !found {
			return fmt.Errorf("failed to extract key '%s' from event", key)
		}
	}
	return nil
}
