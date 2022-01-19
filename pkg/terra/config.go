package terra

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var DefaultConfigSet = ConfigSet{
	// ~8s per block, so ~80s until we give up on the tx getting confirmed
	// Anecdotally it appears anything more than 4 blocks would be an extremely long wait.
	BlocksUntilTxTimeout:  10,
	ConfirmMaxPolls:       100,
	ConfirmPollPeriod:     time.Second,
	FallbackGasPriceULuna: sdk.MustNewDecFromStr("0.01"),
	GasLimitMultiplier:    1.5,
	// The max gas limit per block is 1_000_000_000
	// https://github.com/terra-money/core/blob/d6037b9a12c8bf6b09fe861c8ad93456aac5eebb/app/legacy/migrate.go#L69.
	// The max msg size is 10KB https://github.com/terra-money/core/blob/d6037b9a12c8bf6b09fe861c8ad93456aac5eebb/x/wasm/types/params.go#L15.
	// Our msgs are only OCR reports for now, which will not exceed that size.
	// There appears to be no gas limit per tx, only per block, so theoretically
	// we could include 1000 msgs which use up to 1M gas.
	// To be conservative and since the number of messages we'd
	// have in a batch on average roughly correponds to the number of terra ocr jobs we're running (do not expect more than 100),
	// we can set a max msgs per batch of 100.
	MaxMsgsPerBatch: 100,
}

type Config interface {
	BlocksUntilTxTimeout() int64
	ConfirmMaxPolls() int64
	ConfirmPollPeriod() time.Duration
	FallbackGasPriceULuna() sdk.Dec
	GasLimitMultiplier() float64
	MaxMsgsPerBatch() int64
}

// ConfigSet has configuration fields for default sets and testing.
type ConfigSet struct {
	BlocksUntilTxTimeout  int64
	ConfirmMaxPolls       int64
	ConfirmPollPeriod     time.Duration
	FallbackGasPriceULuna sdk.Dec
	GasLimitMultiplier    float64
	MaxMsgsPerBatch       int64
}

// Config returns a Config backed by this ConfigSet.
func (s ConfigSet) Config() Config {
	return &configSet{s}
}

type configSet struct {
	s ConfigSet
}

func (c *configSet) BlocksUntilTxTimeout() int64 {
	return c.s.BlocksUntilTxTimeout
}

func (c *configSet) ConfirmMaxPolls() int64 {
	return c.s.ConfirmMaxPolls
}

func (c *configSet) ConfirmPollPeriod() time.Duration {
	return c.s.ConfirmPollPeriod
}

func (c *configSet) FallbackGasPriceULuna() sdk.Dec {
	return c.s.FallbackGasPriceULuna
}

func (c *configSet) GasLimitMultiplier() float64 {
	return c.s.GasLimitMultiplier
}

func (c *configSet) MaxMsgsPerBatch() int64 {
	return c.s.MaxMsgsPerBatch
}
