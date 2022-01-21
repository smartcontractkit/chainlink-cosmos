package terra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/smartcontractkit/chainlink/core/utils"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
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
	utils.StartStopOnce
	address     sdk.AccAddress
	chainReader client.Reader
	lggr        Logger
	cr          LatestConfigReader
	cfg         Config
	stop, done  chan struct{}

	// cached state
	mu              sync.RWMutex
	ts              time.Time
	configDigest    types.ConfigDigest
	epoch           uint32
	round           uint8
	latestAnswer    *big.Int
	latestTimestamp time.Time
}

func NewMedianContract(address sdk.AccAddress, chainReader client.Reader, lggr Logger, cr LatestConfigReader, cfg Config) *MedianContract {
	return &MedianContract{
		address:     address,
		chainReader: chainReader,
		lggr:        lggr,
		cr:          cr,
		cfg:         cfg,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}
}

func (ct *MedianContract) Start() error {
	return ct.StartOnce("TerraMedianContract", func() error {
		ct.lggr.Debugf("Starting")
		go ct.pollState()
		return nil
	})
}

func (ct *MedianContract) Close() error {
	return ct.StopOnce("TerraMedianContract", func() error {
		ct.lggr.Debugf("Stopping")
		close(ct.stop)
		<-ct.done
		return nil
	})
}

func (ct *MedianContract) pollState() {
	defer close(ct.done)
	tick := time.After(0)
	for {
		select {
		case <-ct.stop:
			return
		case <-tick:
			ctx, cancel := utils.ContextFromChan(ct.stop)
			done := make(chan struct{})
			go func() {
				defer close(done)
				configDigest, epoch, round, latestAnswer, latestTimestamp, err := ct.latestTransmissionDetails(ctx)
				if err != nil {
					ct.lggr.Errorf("Failed to get latest transmission details", "err", err)
					return
				}
				ct.mu.Lock()
				ct.configDigest = configDigest
				ct.epoch = epoch
				ct.round = round
				ct.latestAnswer = latestAnswer
				ct.latestTimestamp = latestTimestamp
				ct.mu.Unlock()
			}()
			select {
			case <-ct.stop:
				cancel()
				// Note: the client does not respect context, so just return instead of waiting.
				// <-done
				return
			case <-done:
				tick = time.After(utils.WithJitter(ct.cfg.OCR2CachePollPeriod()))
			}
		}
	}
}

// LatestTransmissionDetails fetches the latest cached transmission details
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
	ct.mu.RLock()
	ts := ct.ts
	configDigest = ct.configDigest
	epoch = ct.epoch
	round = ct.round
	latestAnswer = ct.latestAnswer
	latestTimestamp = ct.latestTimestamp
	ct.mu.RUnlock()
	if ts.IsZero() {
		err = errors.New("contract not yet initialized")
	} else if since := time.Since(ts); since > ct.cfg.OCR2CacheTTL() {
		err = fmt.Errorf("failed to get latest transmission details: stale value cached %s ago", since)
	}
	return
}

// latestTransmissionDetails fetches the latest transmission details from address state
func (ct *MedianContract) latestTransmissionDetails(
	ctx context.Context,
) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	latestAnswer *big.Int,
	latestTimestamp time.Time,
	err error,
) {
	resp, err := ct.chainReader.ContractStore(ct.address, []byte(`"latest_transmission_details"`))
	if err != nil {
		// TODO: Verify if this is still necessary
		// https://github.com/smartcontractkit/chainlink-terra/issues/23
		// Handle the 500 error that occurs when there has not been a submission
		// "rpc error: code = Unknown desc = ocr2::state::Transmission not found: address query failed"
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

// LatestRoundRequested fetches the latest round requested from the cache
func (ct *MedianContract) LatestRoundRequested(ctx context.Context, lookback time.Duration) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	err error,
) {
	ct.mu.RLock()
	ts := ct.ts
	configDigest = ct.configDigest
	epoch = ct.epoch
	round = ct.round
	ct.mu.Unlock()
	if ts.IsZero() {
		err = errors.New("contract not yet initialized")
	} else if since := time.Since(ts); since > ct.cfg.OCR2CacheTTL() {
		err = fmt.Errorf("failed to get latest round requested: stale value cached %s ago", since)
	}
	return
}

// latestRoundRequested fetches the latest round requested by filtering event logs
func (ct *MedianContract) latestRoundRequested(ctx context.Context, lookback time.Duration) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	err error,
) {
	// calculate start block
	latestBlock, blkErr := ct.chainReader.LatestBlock()
	if blkErr != nil {
		err = blkErr
		return
	}
	blockNum := uint64(latestBlock.Block.Header.Height) - uint64(lookback/ct.cfg.BlockRate())
	res, err := ct.chainReader.TxsEvents([]string{fmt.Sprintf("tx.height>=%d", blockNum+1), fmt.Sprintf("wasm-new_round.contract_address='%s'", ct.address.String())})
	if err != nil {
		return
	}
	if len(res.TxResponses) == 0 {
		return
	}
	if len(res.TxResponses[0].Logs) == 0 {
		err = fmt.Errorf("No logs found for tx %s", res.TxResponses[0].TxHash)
		return
	}
	// First tx is the latest.
	if len(res.TxResponses[0].Logs[0].Events) == 0 {
		err = fmt.Errorf("No events found for tx %s", res.TxResponses[0].TxHash)
		return
	}

	for _, event := range res.TxResponses[0].Logs[0].Events {
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
