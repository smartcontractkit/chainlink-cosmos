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

	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	cosmosQuery "github.com/cosmos/cosmos-sdk/types/query"
	cosmosTx "github.com/cosmos/cosmos-sdk/types/tx"
	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	pkgTerra "github.com/smartcontractkit/chainlink-terra/pkg/terra"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"go.uber.org/multierr"
)

// NewEnvelopeSourceFactory build a new object that reads observations and
// configurations from the Terra chain.
func NewEnvelopeSourceFactory(client ChainReader, log relayMonitoring.Logger) relayMonitoring.SourceFactory {
	return &envelopeSourceFactory{client, log}
}

type envelopeSourceFactory struct {
	client ChainReader
	log    relayMonitoring.Logger
}

func (e *envelopeSourceFactory) NewSource(
	chainConfig relayMonitoring.ChainConfig,
	feedConfig relayMonitoring.FeedConfig,
) (relayMonitoring.Source, error) {
	terraConfig, ok := chainConfig.(TerraConfig)
	if !ok {
		return nil, fmt.Errorf("expected chainConfig to be of type TerraConfig not %T", chainConfig)
	}
	terraFeedConfig, ok := feedConfig.(TerraFeedConfig)
	if !ok {
		return nil, fmt.Errorf("expected feedConfig to be of type TerraFeedConfig not %T", feedConfig)
	}
	return &envelopeSource{
		e.client,
		e.log,
		terraConfig,
		terraFeedConfig,
	}, nil
}

func (e *envelopeSourceFactory) GetType() string {
	return "envelope"
}

type envelopeSource struct {
	client          ChainReader
	log             relayMonitoring.Logger
	terraConfig     TerraConfig
	terraFeedConfig TerraFeedConfig
}

type linkBalanceResponse struct {
	Balance string `json:"balance"`
}

func (e *envelopeSource) Fetch(ctx context.Context) (interface{}, error) {
	envelope := relayMonitoring.Envelope{}
	var envelopeErr error
	envelopeMu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		configDigest, epoch, round, latestAnswer, latestTimestamp, blockNumber,
			transmitter, aggregatorRoundID, juelsPerFeeCoin, err := e.fetchLatestTransmission(ctx)
		envelopeMu.Lock()
		defer envelopeMu.Unlock()
		if err != nil {
			envelopeErr = multierr.Combine(envelopeErr, fmt.Errorf("failed to fetch transmission: %w", err))
			return
		}
		envelope.ConfigDigest = configDigest
		envelope.Epoch = epoch
		envelope.Round = round
		envelope.LatestAnswer = latestAnswer
		envelope.LatestTimestamp = latestTimestamp
		// Note: block number is read from the transmission transaction, not set_config!
		envelope.BlockNumber = blockNumber
		envelope.Transmitter = transmitter
		envelope.JuelsPerFeeCoin = juelsPerFeeCoin
		envelope.AggregatorRoundID = aggregatorRoundID
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
	wg.Wait()
	return envelope, envelopeErr
}

func (e *envelopeSource) fetchLatestTransmission(ctx context.Context) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	latestAnswer *big.Int,
	latestTimestamp time.Time,
	blockNumber uint64,
	transmitter types.Account,
	aggregatorRoundID uint32,
	juelsPerFeeCoin *big.Int,
	err error,
) {
	query := []string{
		fmt.Sprintf(`wasm-new_transmission.contract_address='%s'`, e.terraFeedConfig.ContractAddressBech32),
	}
	res, err := e.client.TxsEvents(ctx, query, &cosmosQuery.PageRequest{Limit: 1})
	if err != nil {
		return types.ConfigDigest{}, 0, 0, nil, time.Time{}, 0, "", 0, nil,
			fmt.Errorf("failed to fetch latest 'new_transmission' event: %w", err)
	}
	err = e.extractDataFromTxResponse("wasm-new_transmission", e.terraFeedConfig.ContractAddressBech32, res, map[string]func(string) error{
		"config_digest": func(value string) error {
			return pkgTerra.HexToConfigDigest(value, &configDigest)
		},
		"epoch": func(value string) error {
			rawEpoch, parseErr := strconv.ParseUint(value, 10, 32)
			epoch = uint32(rawEpoch)
			return parseErr
		},
		"round": func(value string) error {
			rawRound, parseErr := strconv.ParseUint(value, 10, 8)
			round = uint8(rawRound)
			return parseErr
		},
		"answer": func(value string) error {
			var success bool
			latestAnswer, success = new(big.Int).SetString(value, 10)
			if !success {
				return fmt.Errorf("failed to read latest answer from value '%s'", value)
			}
			return nil
		},
		"observations_timestamp": func(value string) error {
			rawTimestamp, parseErr := strconv.ParseInt(value, 10, 64)
			latestTimestamp = time.Unix(rawTimestamp, 0)
			return parseErr
		},
		"transmitter": func(value string) error {
			transmitter = types.Account(value)
			return nil
		},
		"aggregator_round_id": func(value string) error {
			raw, pasrseErr := strconv.ParseUint(value, 10, 32)
			aggregatorRoundID = uint32(raw)
			return pasrseErr
		},
		"juels_per_fee_coin": func(value string) error {
			var success bool
			juelsPerFeeCoin, success = new(big.Int).SetString(value, 10)
			if !success {
				return fmt.Errorf("failed to parse juel per fee coin from '%s'", value)
			}
			return nil
		},
	})
	if err != nil {
		return types.ConfigDigest{}, 0, 0, nil, time.Time{}, 0, "", 0, nil,
			fmt.Errorf("failed to extract transmission from logs: %w", err)
	}
	blockNumber = uint64(res.TxResponses[0].Height)
	return configDigest, epoch, round, latestAnswer, latestTimestamp, blockNumber,
		transmitter, aggregatorRoundID, juelsPerFeeCoin, nil
}

func (e *envelopeSource) fetchLatestConfig(ctx context.Context) (types.ContractConfig, error) {
	query := []string{
		fmt.Sprintf(`wasm-set_config.contract_address='%s'`, e.terraFeedConfig.ContractAddressBech32),
	}
	res, err := e.client.TxsEvents(ctx, query, &cosmosQuery.PageRequest{Limit: 1})
	if err != nil {
		return types.ContractConfig{}, fmt.Errorf("failed to fetch latest 'set_config' event: %w", err)
	}
	output := types.ContractConfig{}
	err = e.extractDataFromTxResponse("wasm-set_config", e.terraFeedConfig.ContractAddressBech32, res, map[string]func(string) error{
		"latest_config_digest": func(value string) error {
			// parse byte array encoded as hex string
			return pkgTerra.HexToConfigDigest(value, &output.ConfigDigest)
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
			convertErr := pkgTerra.HexToByteArray(value, &v)
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

func (e *envelopeSource) fetchLinkBalance(ctx context.Context) (*big.Int, error) {
	query := fmt.Sprintf(`{"balance":{"address":"%s"}}`, e.terraFeedConfig.ContractAddressBech32)
	res, err := e.client.ContractStore(
		ctx,
		e.terraConfig.LinkTokenAddress,
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

// Helpers

func (e *envelopeSource) extractDataFromTxResponse(
	eventType string,
	contractAddressBech32 string,
	res *cosmosTx.GetTxsEventResponse,
	extractors map[string]func(string) error,
) error {
	if len(res.TxResponses) == 0 ||
		len(res.TxResponses[0].Logs) == 0 ||
		len(res.TxResponses[0].Logs[0].Events) == 0 {
		return fmt.Errorf("%d events found in response", len(res.TxResponses[0].Logs[0].Events))
	}
	// Extract matching events
	events := extractMatchingEvents(res.TxResponses[0].Logs[0].Events, eventType, contractAddressBech32)
	if len(events) == 0 {
		return fmt.Errorf("no event found with type='%s' and contract_address='%s'", eventType, contractAddressBech32)
	}
	if len(events) != 1 {
		e.log.Infow("multiple matching events found, selecting the last one", "type", eventType, "contract_address", contractAddressBech32)
	}
	event := events[len(events)-1]
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

func extractMatchingEvents(events cosmosTypes.StringEvents, eventType, contractAddressBech32 string) []cosmosTypes.StringEvent {
	out := []cosmosTypes.StringEvent{}
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		isMatchingContractAddress := false
		for _, attribute := range event.Attributes {
			if attribute.Key == "contract_address" && attribute.Value == contractAddressBech32 {
				isMatchingContractAddress = true
			}
		}
		if isMatchingContractAddress {
			out = append(out, event)
		}
	}
	return out
}

func checkEventAttributes(
	event cosmosTypes.StringEvent,
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
