package config

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"

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

func ptr[T any](t T) *T {
	return &t
}
