package config

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"github.com/smartcontractkit/chainlink-relay/pkg/utils"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/db"
)

func TestChain_SetFromDB(t *testing.T) {
	gasPriceUAtom := decimal.RequireFromString("0.015")
	gasLimitMultiplier := decimal.RequireFromString("1.5")
	for _, tt := range []struct {
		name  string
		dbCfg *db.ChainCfg
		exp   Chain
	}{
		{"nil", nil, Chain{}},
		{"empty", &db.ChainCfg{}, Chain{}},
		{"full", &db.ChainCfg{
			BlockRate:             utils.MustNewDuration(6 * time.Second),
			BlocksUntilTxTimeout:  null.IntFrom(30),
			ConfirmPollPeriod:     utils.MustNewDuration(time.Second),
			FallbackGasPriceUAtom: null.StringFrom("0.015"),
			FCDURL:                null.StringFrom("http://fake.test"),
			GasLimitMultiplier:    null.FloatFrom(1.5),
			MaxMsgsPerBatch:       null.IntFrom(100),
			OCR2CachePollPeriod:   utils.MustNewDuration(4 * time.Second),
			OCR2CacheTTL:          utils.MustNewDuration(time.Minute),
			TxMsgTimeout:          utils.MustNewDuration(10 * time.Minute),
		}, Chain{
			BlockRate:             utils.MustNewDuration(6 * time.Second),
			BlocksUntilTxTimeout:  ptr[int64](30),
			ConfirmPollPeriod:     utils.MustNewDuration(time.Second),
			FallbackGasPriceUAtom: &gasPriceUAtom,
			FCDURL:                utils.MustParseURL("http://fake.test"),
			GasLimitMultiplier:    &gasLimitMultiplier,
			MaxMsgsPerBatch:       ptr[int64](100),
			OCR2CachePollPeriod:   utils.MustNewDuration(4 * time.Second),
			OCR2CacheTTL:          utils.MustNewDuration(time.Minute),
			TxMsgTimeout:          utils.MustNewDuration(10 * time.Minute),
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var c Chain
			require.NoError(t, c.SetFromDB(tt.dbCfg))
			assert.Equal(t, tt.exp, c)
		})
	}
}

func TestNode_SetFromDB(t *testing.T) {
	for _, tt := range []struct {
		name   string
		dbNode db.Node
		exp    Node
		expErr bool
	}{
		{"empty", db.Node{}, Node{}, false},
		{"url", db.Node{
			Name:          "test-name",
			TendermintURL: "http://fake.test",
		}, Node{
			Name:          ptr("test-name"),
			TendermintURL: utils.MustParseURL("http://fake.test"),
		}, false},
		{"url-missing", db.Node{
			Name: "test-name",
		}, Node{
			Name: ptr("test-name"),
		}, false},
		{"url-invalid", db.Node{
			Name:          "test-name",
			TendermintURL: "asdf;lk.asdf.;lk://asdlkvpoicx;",
		}, Node{}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var n Node
			err := n.SetFromDB(tt.dbNode)
			if tt.expErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.exp, n)
			}
		})
	}
}

func TestConfigDBUpdate(t *testing.T) {
	def := defaultConfigSet

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

func ptr[T any](t T) *T {
	return &t
}
