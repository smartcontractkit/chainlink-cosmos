package terra

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/guregu/null.v4"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/db"
	"github.com/smartcontractkit/chainlink-terra/pkg/terra/mocks"
)

func TestConfig(t *testing.T) {
	def := DefaultConfigSet

	lggr := new(mocks.Logger)
	lggr.On("Warnf", mock.Anything, "FallbackGasPriceULuna", "not-a-number", mock.Anything, mock.Anything).Once()
	cfg := NewConfig(db.ChainCfg{}, def, lggr)
	assert.Equal(t, def.BlocksUntilTxTimeout, cfg.BlocksUntilTxTimeout())
	assert.Equal(t, def.ConfirmPollPeriod, cfg.ConfirmPollPeriod())
	assert.Equal(t, def.FallbackGasPriceULuna, cfg.FallbackGasPriceULuna())
	assert.Equal(t, def.GasLimitMultiplier, cfg.GasLimitMultiplier())
	assert.Equal(t, def.MaxMsgsPerBatch, cfg.MaxMsgsPerBatch())

	updated := db.ChainCfg{
		BlocksUntilTxTimeout:  null.IntFrom(1000),
		FallbackGasPriceULuna: null.StringFrom("5.6"),
	}
	cfg.Update(updated)
	assert.Equal(t, updated.BlocksUntilTxTimeout.Int64, cfg.BlocksUntilTxTimeout())
	assert.Equal(t, def.ConfirmPollPeriod, cfg.ConfirmPollPeriod())
	assert.Equal(t, sdk.MustNewDecFromStr(updated.FallbackGasPriceULuna.String), cfg.FallbackGasPriceULuna())
	assert.Equal(t, def.GasLimitMultiplier, cfg.GasLimitMultiplier())
	assert.Equal(t, def.MaxMsgsPerBatch, cfg.MaxMsgsPerBatch())

	updated = db.ChainCfg{
		FallbackGasPriceULuna: null.StringFrom("not-a-number"),
	}
	cfg.Update(updated)
	assert.Equal(t, def.FallbackGasPriceULuna, cfg.FallbackGasPriceULuna())
}
