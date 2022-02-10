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

	cosmosQuery "github.com/cosmos/cosmos-sdk/types/query"
	cosmosTx "github.com/cosmos/cosmos-sdk/types/tx"
	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	pkgTerra "github.com/smartcontractkit/chainlink-terra/pkg/terra"
	pkgClient "github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
	"github.com/smartcontractkit/chainlink/core/logger"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"go.uber.org/multierr"
)

// NewEnvelopeSourceFactory build a new object that reads observations and
// configurations from the Terra chain.
func NewEnvelopeSourceFactory(client pkgClient.Reader, log logger.Logger) relayMonitoring.SourceFactory {
	return &envelopeSourceFactory{client, log}
}

type envelopeSourceFactory struct {
	client pkgClient.Reader
	log    logger.Logger
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

type envelopeSource struct {
	client          pkgClient.Reader
	log             logger.Logger
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
		configDigest, epoch, round, latestAnswer, latestTimestamp, blockNumber, transmitter, aggregatorRoundID, juelsPerFeeCoin, err := e.fetchLatestTransmission()
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
		envelope.BlockNumber = blockNumber
		envelope.Transmitter = transmitter
		envelope.JuelsPerFeeCoin = juelsPerFeeCoin
		envelope.AggregatorRoundID = aggregatorRoundID
	}()
	go func() {
		defer wg.Done()
		contractConfig, err := e.fetchLatestConfig()
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
		query := fmt.Sprintf(`{"balance":{"address":"%s"}}`, e.terraFeedConfig.ContractAddressBech32)
		res, err := e.client.ContractStore(
			e.terraConfig.LinkTokenAddress,
			[]byte(query),
		)
		envelopeMu.Lock()
		defer envelopeMu.Unlock()
		if err != nil {
			envelopeErr = multierr.Combine(envelopeErr, fmt.Errorf("failed to fetch balance: %w", err))
			return
		}
		balanceRes := linkBalanceResponse{}
		if err = json.Unmarshal(res, &balanceRes); err != nil {
			envelopeErr = multierr.Combine(envelopeErr, fmt.Errorf("failed to unmarshal balance response: %w", err))
			return
		}
		balance, err := strconv.ParseUint(balanceRes.Balance, 10, 64)
		if err != nil {
			envelopeErr = multierr.Combine(envelopeErr, fmt.Errorf("failed to parse uint64 balance from '%s': %w", balanceRes.Balance, err))
			return
		}
		envelope.LinkBalance = balance
	}()
	wg.Wait()
	return envelope, envelopeErr
}

func (e *envelopeSource) fetchLatestTransmission() (
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
		fmt.Sprintf("wasm-new_transmission.contract_address='%s'", e.terraFeedConfig.ContractAddressBech32),
	}
	res, err := e.client.TxsEvents(query, &cosmosQuery.PageRequest{Limit: 1})
	if err != nil {
		return types.ConfigDigest{}, 0, 0, nil, time.Time{}, 0, "", 0, nil,
			fmt.Errorf("failed to fetch latest 'new_transmission' event: %w", err)
	}
	err = extractDataFromTxResponse("wasm-new_transmission", res, map[string]func(string) error{
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
		"juels_per_luna": func(value string) error {
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

func (e *envelopeSource) fetchLatestConfig() (types.ContractConfig, error) {
	query := []string{
		fmt.Sprintf("wasm-set_config.contract_address='%s'", e.terraFeedConfig.ContractAddressBech32),
	}
	res, err := e.client.TxsEvents(query, &cosmosQuery.PageRequest{Limit: 1})
	if err != nil {
		return types.ContractConfig{}, fmt.Errorf("failed to fetch latest 'set_config' event: %w", err)
	}
	output := types.ContractConfig{}
	err = extractDataFromTxResponse("wasm-set_config", res, map[string]func(string) error{
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

// Helpers

func extractDataFromTxResponse(eventType string, res *cosmosTx.GetTxsEventResponse, extractors map[string]func(string) error) error {
	if len(res.TxResponses) == 0 ||
		len(res.TxResponses[0].Logs) == 0 ||
		len(res.TxResponses[0].Logs[0].Events) == 0 {
		return fmt.Errorf("no events found of event type '%s'", eventType)
	}
	for _, event := range res.TxResponses[0].Logs[0].Events {
		if event.Type != eventType {
			continue
		}
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
	}
	return nil
}
