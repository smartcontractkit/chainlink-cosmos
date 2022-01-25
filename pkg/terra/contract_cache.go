package terra

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/smartcontractkit/chainlink/core/utils"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

var _ median.MedianContract = (*ContractCache)(nil)

type ContractCache struct {
	cfg    Config
	reader *OCR2Reader
	lggr   Logger

	stop, done chan struct{}

	configMu    sync.RWMutex
	configTS    time.Time
	configBlock uint64
	config      types.ContractConfig

	transMu         sync.RWMutex
	transTS         time.Time
	digest          types.ConfigDigest
	epoch           uint32
	round           uint8
	latestAnswer    *big.Int
	latestTimestamp time.Time
}

func NewContractCache(cfg Config, reader *OCR2Reader, lggr Logger) *ContractCache {
	return &ContractCache{
		cfg:    cfg,
		reader: reader,
		lggr:   lggr,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (cc *ContractCache) Start() error {
	go cc.poll()
	return nil
}

func (cc *ContractCache) Close() error {
	close(cc.stop)
	select {
	case <-time.After(time.Second):
		// can't rely on clients to cancel
	case <-cc.done:
	}
	return nil
}

func (cc *ContractCache) poll() {
	defer close(cc.done)
	tick := time.After(0)
	for {
		select {
		case <-cc.stop:
			return
		case <-tick:
			ctx, _ := utils.ContextFromChan(cc.stop)
			if err := cc.updateConfig(ctx); err != nil {
				cc.lggr.Errorf("Failed to update config", "err", err)
			}
			if ctx.Err() != nil { // b/c client doesn't use ctx
				return
			}
			if err := cc.updateTransmission(ctx); err != nil {
				cc.lggr.Errorf("Failed to update transmission", "err", err)
			}
			tick = time.After(utils.WithJitter(cc.cfg.OCR2CachePollPeriod()))
		}
	}
}

func (cc *ContractCache) updateConfig(ctx context.Context) error {
	changedInBlock, configDigest, err := cc.reader.LatestConfigDetails(ctx)
	if err != nil {
		return errors.Wrap(err, "fetch latest config details")
	}
	if err = ctx.Err(); err != nil { // b/c client doesn't use ctx
		return err
	}
	now := time.Now()
	cc.configMu.Lock()
	same := cc.configBlock == changedInBlock && cc.config.ConfigDigest == configDigest
	if same {
		cc.configTS = now // refresh TTL
	}
	cc.configMu.Unlock()
	if same {
		return nil
	}
	contractConfig, err := cc.reader.LatestConfig(ctx, changedInBlock)
	if err != nil {
		return errors.Wrapf(err, "fetch latest config, block %d", changedInBlock)
	}
	now = time.Now()
	cc.configMu.Lock()
	cc.configTS = now
	cc.configBlock = changedInBlock
	cc.config = contractConfig
	cc.configMu.Unlock()
	return nil
}

func (cc *ContractCache) updateTransmission(ctx context.Context) error {
	digest, epoch, round, latestAnswer, latestTimestamp, err := cc.reader.LatestTransmissionDetails(ctx)
	if err != nil {
		return errors.Wrap(err, "fetch latest transmission")
	}
	now := time.Now()
	cc.transMu.Lock()
	cc.transTS = now
	cc.digest = digest
	cc.epoch = epoch
	cc.round = round
	cc.latestAnswer = latestAnswer
	cc.latestTimestamp = latestTimestamp
	cc.transMu.Unlock()
	return nil
}

func (cc *ContractCache) checkTS(ts time.Time) error {
	if ts.IsZero() {
		return errors.New("contract cache not yet initialized")
	} else if since := time.Since(ts); since > cc.cfg.OCR2CacheTTL() {
		return fmt.Errorf("contract cache expired: value cached %s ago", since)
	}
	return nil
}

func (cc *ContractCache) LatestConfigDetails(ctx context.Context) (changedInBlock uint64, configDigest types.ConfigDigest, err error) {
	cc.configMu.RLock()
	ts := cc.configTS
	changedInBlock = cc.configBlock
	configDigest = cc.config.ConfigDigest
	cc.configMu.RUnlock()
	err = cc.checkTS(ts)
	return
}

func (cc *ContractCache) LatestConfig(ctx context.Context, changedInBlock uint64) (contractConfig types.ContractConfig, err error) {
	cc.configMu.RLock()
	ts := cc.configTS
	contractConfig = cc.config
	cachedBlock := cc.configBlock
	cc.configMu.RUnlock()
	err = cc.checkTS(ts)
	if err == nil && cachedBlock != changedInBlock {
		err = fmt.Errorf("failed to get config from %d: latest config in cache is from %d", changedInBlock, cachedBlock)
	}
	return
}

func (cc *ContractCache) LatestTransmissionDetails(ctx context.Context) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	latestAnswer *big.Int,
	latestTimestamp time.Time,
	err error,
) {
	cc.transMu.RLock()
	ts := cc.transTS
	configDigest = cc.digest
	epoch = cc.epoch
	round = cc.round
	latestAnswer = cc.latestAnswer
	latestTimestamp = cc.latestTimestamp
	cc.transMu.RUnlock()
	err = cc.checkTS(ts)
	return
}

func (cc *ContractCache) LatestRoundRequested(ctx context.Context, lookback time.Duration) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	err error,
) {
	cc.transMu.RLock()
	ts := cc.transTS
	configDigest = cc.digest
	epoch = cc.epoch
	round = cc.round
	cc.transMu.RUnlock()
	err = cc.checkTS(ts)
	return
}
