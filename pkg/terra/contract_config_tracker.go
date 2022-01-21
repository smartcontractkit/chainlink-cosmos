package terra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/smartcontractkit/chainlink/core/utils"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
)

var _ types.ContractConfigTracker = (*ContractTracker)(nil)

type ContractTracker struct {
	utils.StartStopOnce
	jobID       string
	address     sdk.AccAddress
	chainReader client.Reader
	log         Logger
	cfg         Config
	stop, done  chan struct{}

	// cached state
	mu             sync.RWMutex
	ts             *time.Time
	changedInBlock uint64
	configDigest   types.ConfigDigest
	contractConfig types.ContractConfig
}

func NewContractTracker(address sdk.AccAddress, jobID string, chainReader client.Reader, cfg Config, lggr Logger) *ContractTracker {
	contract := ContractTracker{
		jobID:       jobID,
		address:     address,
		chainReader: chainReader,
		log:         lggr,
		cfg:         cfg,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}
	return &contract
}

func (ct *ContractTracker) Start() error {
	return ct.StartOnce("TerraContractTracker", func() error {
		ct.log.Debugf("Starting")
		go ct.pollState()
		return nil
	})
}

func (ct *ContractTracker) Close() error {
	return ct.StopOnce("TerraContractTracker", func() error {
		ct.log.Debugf("Stopping")
		close(ct.stop)
		<-ct.done
		return nil
	})
}

func (ct *ContractTracker) pollState() {
	defer close(ct.done)
	tick := time.After(utils.WithJitter(ct.cfg.ConfirmPollPeriod()))
	for {
		select {
		case <-ct.stop:
			return
		case <-tick:
			ctx, cancel := utils.ContextFromChan(ct.stop)
			done := make(chan struct{})
			go func() {
				defer close(done)
				changedInBlock, configDigest, err := ct.latestConfigDetails(ctx)
				if err != nil {
					ct.log.Errorf("Failed to get latest config details", "err", err)
					return
				}
				ct.mu.RLock()
				update := ct.changedInBlock != ct.changedInBlock
				ct.mu.RUnlock()
				if !update {
					// no change
					return
				}
				contractConfig, err := ct.latestConfig(ctx, changedInBlock)
				if err != nil {
					ct.log.Errorf("Failed to get latest config", "block", changedInBlock, "err", err)
					return
				}
				now := time.Now()
				//TODO sanity check digest/block?
				ct.mu.Lock()
				ct.ts = &now
				ct.changedInBlock = changedInBlock
				ct.configDigest = configDigest
				ct.contractConfig = contractConfig
				ct.mu.Unlock()
			}()
			select {
			case <-ct.stop:
				cancel()
				// Note: the client does not respect context, so just return instead of waiting.
				// <-done
				return
			case <-done:
				tick = time.After(utils.WithJitter(ct.cfg.ConfirmPollPeriod()))
			}
		}
	}
}

// Unused, libocr will use polling
func (ct *ContractTracker) Notify() <-chan struct{} {
	return nil
}

// LatestConfigDetails returns data by reading the address state and is called when Notify is triggered or the config poll timer is triggered
func (ct *ContractTracker) LatestConfigDetails(ctx context.Context) (changedInBlock uint64, configDigest types.ConfigDigest, err error) {
	ct.mu.RLock()
	ts := ct.ts
	changedInBlock = ct.changedInBlock
	configDigest = ct.configDigest
	ct.mu.RUnlock()
	if ts == nil {
		err = errors.New("config details not yet initialized")
	} else if since := time.Since(*ts); since > ct.cfg.OCRCacheTTL() {
		err = fmt.Errorf("failed to get config details: stale value cached %s ago", since)
	}
	return
}

func (ct *ContractTracker) latestConfigDetails(ctx context.Context) (changedInBlock uint64, configDigest types.ConfigDigest, err error) {
	resp, err := ct.chainReader.ContractStore(
		ct.address,
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

// LatestConfig returns data by searching emitted events and is called in the same scenario as LatestConfigDetails
func (ct *ContractTracker) LatestConfig(_ context.Context, changedInBlock uint64) (contractConfig types.ContractConfig, err error) {
	ct.mu.RLock()
	ts := ct.ts
	contractConfig = ct.contractConfig
	cachedBlock := ct.changedInBlock
	ct.mu.RUnlock()
	if ts == nil {
		err = errors.New("config not yet initialized")
	} else if cachedBlock != changedInBlock {
		err = fmt.Errorf("failed to get config from %d: latest config in cache is from %d", changedInBlock, cachedBlock)
	}
	return
}

func (ct *ContractTracker) latestConfig(ctx context.Context, changedInBlock uint64) (types.ContractConfig, error) {
	query := []string{fmt.Sprintf("tx.height=%d", changedInBlock), fmt.Sprintf("wasm-set_config.contract_address='%s'", ct.address)}
	res, err := ct.chainReader.TxsEvents(query)
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

// LatestBlockHeight returns the height of the most recent block in the chain.
func (ct *ContractTracker) LatestBlockHeight(ctx context.Context) (blockHeight uint64, err error) {
	b, err := ct.chainReader.LatestBlock()
	if err != nil {
		return 0, err
	}
	return uint64(b.Block.Header.Height), nil
}
