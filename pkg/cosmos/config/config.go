package config

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/multierr"

	sdk "github.com/cosmos/cosmos-sdk/types"

	relayconfig "github.com/smartcontractkit/chainlink-relay/pkg/config"
	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"github.com/smartcontractkit/chainlink-relay/pkg/utils"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/client"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/db"
)

// Global defaults.
var defaultConfigSet = configSet{
	BlockRate: 6 * time.Second,
	// ~6s per block, so ~3m until we give up on the tx getting confirmed
	// Anecdotally it appears anything more than 4 blocks would be an extremely long wait,
	// In practice during the UST depegging and subsequent extreme congestion, we saw
	// ~16 block FIFO lineups.
	BlocksUntilTxTimeout:  30,
	ConfirmPollPeriod:     time.Second,
	FallbackGasPriceUAtom: sdk.MustNewDecFromStr("0.015"),
	// This is high since we simulate before signing the transaction.
	// There's a chicken and egg problem: need to sign to simulate accurately
	// but you need to specify a gas limit when signing.
	// TODO: Determine how much gas a signature adds and then
	// add that directly so we can be more accurate.
	GasLimitMultiplier: client.DefaultGasLimitMultiplier,
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
	TxMsgTimeout:        10 * time.Minute,
}

type Config interface {
	BlockRate() time.Duration
	BlocksUntilTxTimeout() int64
	ConfirmPollPeriod() time.Duration
	FallbackGasPriceUAtom() sdk.Dec
	LCDURL() url.URL
	GasLimitMultiplier() float64
	MaxMsgsPerBatch() int64
	OCR2CachePollPeriod() time.Duration
	OCR2CacheTTL() time.Duration
	TxMsgTimeout() time.Duration
}

type configSet struct {
	BlockRate             time.Duration
	BlocksUntilTxTimeout  int64
	ConfirmPollPeriod     time.Duration
	FallbackGasPriceUAtom sdk.Dec
	LCDURL                url.URL
	GasLimitMultiplier    float64
	MaxMsgsPerBatch       int64
	OCR2CachePollPeriod   time.Duration
	OCR2CacheTTL          time.Duration
	TxMsgTimeout          time.Duration
}

var _ Config = (*config)(nil)

type config struct {
	defaults configSet
	chain    db.ChainCfg
	chainMu  sync.RWMutex
	lggr     logger.Logger
}

// NewConfig returns a Config with defaults overridden by dbcfg.
// TODO: remove mutex
func NewConfig(dbcfg db.ChainCfg, lggr logger.Logger) *config {
	return &config{
		defaults: defaultConfigSet,
		chain:    dbcfg,
		lggr:     lggr,
	}
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

func (c *config) FallbackGasPriceUAtom() sdk.Dec {
	c.chainMu.RLock()
	ch := c.chain.FallbackGasPriceUAtom
	c.chainMu.RUnlock()
	if ch.Valid {
		str := ch.String
		dec, err := sdk.NewDecFromStr(str)
		if err == nil {
			return dec
		}
		c.lggr.Warnf(invalidFallbackMsg, "FallbackGasPriceUAtom", str, c.defaults.FallbackGasPriceUAtom, err)
	}
	return c.defaults.FallbackGasPriceUAtom
}

func (c *config) LCDURL() url.URL {
	c.chainMu.RLock()
	ch := c.chain.LCDURL
	c.chainMu.RUnlock()
	if ch.Valid {
		str := ch.String
		u, err := url.Parse(str)
		if err == nil {
			return *u
		}
		c.lggr.Warnf(invalidFallbackMsg, "LCDURL", str, c.defaults.LCDURL, err)
	}
	return c.defaults.LCDURL
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

func (c *config) TxMsgTimeout() time.Duration {
	c.chainMu.RLock()
	ch := c.chain.TxMsgTimeout
	c.chainMu.RUnlock()
	if ch != nil {
		return ch.Duration()
	}
	return c.defaults.TxMsgTimeout
}

const invalidFallbackMsg = `Invalid value provided for %s, "%s" - falling back to default "%s": %v`

type Chain struct {
	BlockRate             *utils.Duration
	BlocksUntilTxTimeout  *int64
	ConfirmPollPeriod     *utils.Duration
	FallbackGasPriceUAtom *decimal.Decimal
	LCDURL                *utils.URL
	GasLimitMultiplier    *decimal.Decimal
	MaxMsgsPerBatch       *int64
	OCR2CachePollPeriod   *utils.Duration
	OCR2CacheTTL          *utils.Duration
	TxMsgTimeout          *utils.Duration
}

func (c *Chain) SetFromDB(cfg *db.ChainCfg) error {
	if cfg == nil {
		return nil
	}
	if cfg.BlockRate != nil {
		c.BlockRate = utils.MustNewDuration(cfg.BlockRate.Duration())
	}
	if cfg.BlocksUntilTxTimeout.Valid {
		c.BlocksUntilTxTimeout = &cfg.BlocksUntilTxTimeout.Int64
	}
	if cfg.ConfirmPollPeriod != nil {
		c.ConfirmPollPeriod = utils.MustNewDuration(cfg.ConfirmPollPeriod.Duration())
	}
	if cfg.FallbackGasPriceUAtom.Valid {
		s := cfg.FallbackGasPriceUAtom.String
		d, err := decimal.NewFromString(s)
		if err != nil {
			return fmt.Errorf("invalid decimal FallbackGasPriceUAtom: %s", s)
		}
		c.FallbackGasPriceUAtom = &d
	}
	if cfg.LCDURL.Valid {
		s := cfg.LCDURL.String
		d, err := url.Parse(s)
		if err != nil {
			return fmt.Errorf("invalid LCDURL: %s", s)
		}
		c.LCDURL = (*utils.URL)(d)
	}
	if cfg.GasLimitMultiplier.Valid {
		d := decimal.NewFromFloat(cfg.GasLimitMultiplier.Float64)
		c.GasLimitMultiplier = &d
	}
	if cfg.MaxMsgsPerBatch.Valid {
		c.MaxMsgsPerBatch = &cfg.MaxMsgsPerBatch.Int64
	}
	if cfg.OCR2CachePollPeriod != nil {
		c.OCR2CachePollPeriod = utils.MustNewDuration(cfg.OCR2CachePollPeriod.Duration())
	}
	if cfg.OCR2CacheTTL != nil {
		c.OCR2CacheTTL = utils.MustNewDuration(cfg.OCR2CacheTTL.Duration())
	}
	if cfg.TxMsgTimeout != nil {
		c.TxMsgTimeout = utils.MustNewDuration(cfg.TxMsgTimeout.Duration())
	}
	return nil
}

func (c *Chain) SetDefaults() {
	if c.BlockRate == nil {
		c.BlockRate = utils.MustNewDuration(defaultConfigSet.BlockRate)
	}
	if c.BlocksUntilTxTimeout == nil {
		c.BlocksUntilTxTimeout = &defaultConfigSet.BlocksUntilTxTimeout
	}
	if c.ConfirmPollPeriod == nil {
		c.ConfirmPollPeriod = utils.MustNewDuration(defaultConfigSet.ConfirmPollPeriod)
	}
	if c.FallbackGasPriceUAtom == nil {
		d := decimal.NewFromBigInt(defaultConfigSet.FallbackGasPriceUAtom.BigInt(), -sdk.Precision)
		c.FallbackGasPriceUAtom = &d
	}
	if c.LCDURL == nil {
		c.LCDURL = (*utils.URL)(&defaultConfigSet.LCDURL)
	}
	if c.GasLimitMultiplier == nil {
		d := decimal.NewFromFloat(defaultConfigSet.GasLimitMultiplier)
		c.GasLimitMultiplier = &d
	}
	if c.MaxMsgsPerBatch == nil {
		c.MaxMsgsPerBatch = &defaultConfigSet.MaxMsgsPerBatch
	}
	if c.OCR2CachePollPeriod == nil {
		c.OCR2CachePollPeriod = utils.MustNewDuration(defaultConfigSet.OCR2CachePollPeriod)
	}
	if c.OCR2CacheTTL == nil {
		c.OCR2CacheTTL = utils.MustNewDuration(defaultConfigSet.OCR2CacheTTL)
	}
	if c.TxMsgTimeout == nil {
		c.TxMsgTimeout = utils.MustNewDuration(defaultConfigSet.TxMsgTimeout)
	}
}

type Node struct {
	Name          *string
	TendermintURL *utils.URL
}

func (n *Node) SetFromDB(db db.Node) error {
	if db.Name != "" {
		n.Name = &db.Name
	}
	if db.TendermintURL != "" {
		u, err := url.Parse(db.TendermintURL)
		if err != nil {
			return err
		}
		n.TendermintURL = (*utils.URL)(u)
	}
	return nil
}

func (n *Node) ValidateConfig() (err error) {
	if n.Name == nil {
		err = multierr.Append(err, relayconfig.ErrMissing{Name: "Name", Msg: "required for all nodes"})
	} else if *n.Name == "" {
		err = multierr.Append(err, relayconfig.ErrEmpty{Name: "Name", Msg: "required for all nodes"})
	}
	if n.TendermintURL == nil {
		err = multierr.Append(err, relayconfig.ErrMissing{Name: "TendermintURL", Msg: "required for all nodes"})
	}
	return
}
