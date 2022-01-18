package terra

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var DefaultConfigSet = ConfigSet{
	ConfirmMaxPolls:       100,
	ConfirmPollPeriod:     time.Second,
	FallbackGasPriceULuna: sdk.MustNewDecFromStr("0.01"),
	GasLimitMultiplier:    1.5,
}

type Config interface {
	ConfirmMaxPolls() int64
	ConfirmPollPeriod() time.Duration
	FallbackGasPriceULuna() sdk.Dec
	GasLimitMultiplier() float64
}

// ConfigSet has configuration fields for default sets and testing.
type ConfigSet struct {
	ConfirmMaxPolls       int64
	ConfirmPollPeriod     time.Duration
	FallbackGasPriceULuna sdk.Dec
	GasLimitMultiplier    float64
}

// Config returns a Config backed by this ConfigSet.
func (s ConfigSet) Config() Config {
	return &configSet{s}
}

type configSet struct {
	s ConfigSet
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
