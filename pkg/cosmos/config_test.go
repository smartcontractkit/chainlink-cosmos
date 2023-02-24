package cosmos

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"github.com/smartcontractkit/chainlink-relay/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"

	"github.com/smartcontractkit/chainlink-terra/pkg/cosmos/db"
)

func TestConfig(t *testing.T) {
	def := DefaultConfigSet

	lggr, logs := logger.TestObserved(t, zap.WarnLevel)
	cfg := NewConfig(db.ChainCfg{}, lggr)
	assert.Equal(t, def.BlockRate, cfg.BlockRate())
	assert.Equal(t, def.BlocksUntilTxTimeout, cfg.BlocksUntilTxTimeout())
	assert.Equal(t, def.ConfirmPollPeriod, cfg.ConfirmPollPeriod())
	assert.Equal(t, def.FallbackGasPriceUAtom, cfg.FallbackGasPriceUAtom())
	assert.Equal(t, def.FCDURL, cfg.FCDURL())
	assert.Equal(t, def.GasLimitMultiplier, cfg.GasLimitMultiplier())
	assert.Equal(t, def.MaxMsgsPerBatch, cfg.MaxMsgsPerBatch())

	minute, err := utils.NewDuration(time.Minute)
	require.NoError(t, err)
	updated := db.ChainCfg{
		BlockRate:             &minute,
		BlocksUntilTxTimeout:  null.IntFrom(1000),
		FallbackGasPriceUAtom: null.StringFrom("5.6"),
		FCDURL:                null.StringFrom("http://example.com/fcd"),
	}
	cfg.Update(updated)
	assert.Equal(t, updated.BlocksUntilTxTimeout.Int64, cfg.BlocksUntilTxTimeout())
	assert.Equal(t, updated.BlockRate.Duration(), cfg.BlockRate())
	assert.Equal(t, def.ConfirmPollPeriod, cfg.ConfirmPollPeriod())
	assert.Equal(t, sdk.MustNewDecFromStr(updated.FallbackGasPriceUAtom.String), cfg.FallbackGasPriceUAtom())
	fcdURL := cfg.FCDURL()
	assert.Equal(t, updated.FCDURL.String, fcdURL.String())
	assert.Equal(t, def.GasLimitMultiplier, cfg.GasLimitMultiplier())
	assert.Equal(t, def.MaxMsgsPerBatch, cfg.MaxMsgsPerBatch())

	updated = db.ChainCfg{
		FallbackGasPriceUAtom: null.StringFrom("not-a-number"),
	}
	cfg.Update(updated)
	assert.Equal(t, def.FallbackGasPriceUAtom, cfg.FallbackGasPriceUAtom())
	if all := logs.All(); assert.Len(t, all, 1) {
		assert.Contains(t, all[0].Message, `Invalid value provided for FallbackGasPriceUAtom, "not-a-number"`)
	}
}
