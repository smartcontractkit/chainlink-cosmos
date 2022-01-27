package terra

import (
	"net/url"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/db"
)

// Global terra defaults.
var defaultConfigSet = configSet{
	BlockRate: 6 * time.Second,
	// ~6s per block, so ~1m until we give up on the tx getting confirmed
	// Anecdotally it appears anything more than 4 blocks would be an extremely long wait.
	BlocksUntilTxTimeout:  10,
	ConfirmPollPeriod:     time.Second,
	FallbackGasPriceULuna: sdk.MustNewDecFromStr("0.015"),
	GasLimitMultiplier:    1.2,
	// The max gas limit per block is 1_000_000_000
	// https://github.com/terra-money/core/blob/d6037b9a12c8bf6b09fe861c8ad93456aac5eebb/app/legacy/migrate.go#L69.
	// The max msg size is 10KB https://github.com/terra-money/core/blob/d6037b9a12c8bf6b09fe861c8ad93456aac5eebb/x/wasm/types/params.go#L15.
	// Our msgs are only OCR reports for now, which will not exceed that size.
	// There appears to be no gas limit per tx, only per block, so theoretically
	// we could include 1000 msgs which use up to 1M gas.
	// To be conservative and since the number of messages we'd
	// have in a batch on average roughly corresponds to the number of terra ocr jobs we're running (do not expect more than 100),
	// we can set a max msgs per batch of 100.
	MaxMsgsPerBatch:     100,
	OCR2CachePollPeriod: 4 * time.Second,
	OCR2CacheTTL:        time.Minute,
}

type Config interface {
	BlockRate() time.Duration
	BlocksUntilTxTimeout() int64
	ConfirmPollPeriod() time.Duration
	FallbackGasPriceULuna() sdk.Dec
	FCDURL() url.URL
	GasLimitMultiplier() float64
	MaxMsgsPerBatch() int64
	OCR2CachePollPeriod() time.Duration
	OCR2CacheTTL() time.Duration

	// Update sets new chain config values.
	Update(db.ChainCfg)
}

type configSet struct {
	BlockRate             time.Duration
	BlocksUntilTxTimeout  int64
	ConfirmPollPeriod     time.Duration
	FallbackGasPriceULuna sdk.Dec
	FCDURL                url.URL
	GasLimitMultiplier    float64
	MaxMsgsPerBatch       int64
	OCR2CachePollPeriod   time.Duration
	OCR2CacheTTL          time.Duration
}

var _ Config = (*config)(nil)

type config struct {
	defaults configSet
	chain    db.ChainCfg
	chainMu  sync.RWMutex
	lggr     Logger
}

// NewConfig returns a Config with defaults overridden by dbcfg.
func NewConfig(dbcfg db.ChainCfg, lggr Logger) *config {
	return &config{
		defaults: defaultConfigSet,
		chain:    dbcfg,
		lggr:     lggr,
	}
}

func (c *config) Update(dbcfg db.ChainCfg) {
	c.chainMu.Lock()
	c.chain = dbcfg
	c.chainMu.Unlock()
}

func (c *config) BlockRate() time.Duration {
	c.chainMu.RLock()
	ch := c.chain.BlockRate
	c.chainMu.RUnlock()
	if ch != nil {
		return ch.Duration()
	}
	return c.defaults.BlockRate
}

func (c *config) BlocksUntilTxTimeout() int64 {
	c.chainMu.RLock()
	ch := c.chain.BlocksUntilTxTimeout
	c.chainMu.RUnlock()
	if ch.Valid {
		return ch.Int64
	}
	return c.defaults.BlocksUntilTxTimeout
}

func (c *config) ConfirmPollPeriod() time.Duration {
	c.chainMu.RLock()
	ch := c.chain.ConfirmPollPeriod
	c.chainMu.RUnlock()
	if ch != nil {
		return ch.Duration()
	}
	return c.defaults.ConfirmPollPeriod
}

func (c *config) FallbackGasPriceULuna() sdk.Dec {
	c.chainMu.RLock()
	ch := c.chain.FallbackGasPriceULuna
	c.chainMu.RUnlock()
	if ch.Valid {
		str := ch.String
		dec, err := sdk.NewDecFromStr(str)
		if err == nil {
			return dec
		}
		c.lggr.Warnf(invalidFallbackMsg, "FallbackGasPriceULuna", str, c.defaults.FallbackGasPriceULuna, err)
	}
	return c.defaults.FallbackGasPriceULuna
}

func (c *config) FCDURL() url.URL {
	c.chainMu.RLock()
	ch := c.chain.FCDURL
	c.chainMu.RUnlock()
	if ch.Valid {
		str := ch.String
		u, err := url.Parse(str)
		if err == nil {
			return *u
		}
		c.lggr.Warnf(invalidFallbackMsg, "FCDURL", str, c.defaults.FCDURL, err)
	}
	return c.defaults.FCDURL
}

func (c *config) GasLimitMultiplier() float64 {
	c.chainMu.RLock()
	ch := c.chain.GasLimitMultiplier
	c.chainMu.RUnlock()
	if ch.Valid {
		return ch.Float64
	}
	return c.defaults.GasLimitMultiplier
}

func (c *config) MaxMsgsPerBatch() int64 {
	c.chainMu.RLock()
	ch := c.chain.MaxMsgsPerBatch
	c.chainMu.RUnlock()
	if ch.Valid {
		return ch.Int64
	}
	return c.defaults.MaxMsgsPerBatch
}

func (c *config) OCR2CachePollPeriod() time.Duration {
	c.chainMu.RLock()
	ch := c.chain.OCR2CachePollPeriod
	c.chainMu.RUnlock()
	if ch != nil {
		return ch.Duration()
	}
	return c.defaults.OCR2CachePollPeriod
}

func (c *config) OCR2CacheTTL() time.Duration {
	c.chainMu.RLock()
	ch := c.chain.OCR2CacheTTL
	c.chainMu.RUnlock()
	if ch != nil {
		return ch.Duration()
	}
	return c.defaults.OCR2CacheTTL
}

const invalidFallbackMsg = `Invalid value provided for %s, "%s" - falling back to default "%s": %v`
