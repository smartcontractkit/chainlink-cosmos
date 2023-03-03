package client

import (
	"net/url"
	"testing"
	"time"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGasPriceEstimators(t *testing.T) {
	lggr, logs := logger.TestObserved(t, zap.WarnLevel)
	assertLogsLen := func(t *testing.T, l int) func() {
		return func() { assert.Len(t, logs.TakeAll(), l) }
	}

	t.Run("fixed", func(t *testing.T) {
		gpeFixed := NewFixedGasPriceEstimator(map[string]sdk.DecCoin{
			"ucosm": sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("10")),
		})
		p, err := gpeFixed.GasPrices()
		require.NoError(t, err)
		price, ok := p["ucosm"]
		require.True(t, ok)
		assert.Equal(t, "ucosm", price.Denom)
		assert.Equal(t, "10.000000000000000000", price.Amount.String())
	})

	t.Run("caching", func(t *testing.T) {
		gpeFCD := NewFCDGasPriceEstimator(newConfig(t, "https://fcd.terra.dev:443/v1/txs/gas_prices"), 10*time.Second, lggr)
		cachingFCD := NewCachingGasPriceEstimator(gpeFCD, lggr)

		// Fill cache
		prices, err := cachingFCD.GasPrices()
		require.NoError(t, err)

		// Use cache
		const badURL = "https://does.not.exist:443/v1/txs/gas_prices"
		gpeFCD.cfg = newConfig(t, badURL)
		t.Cleanup(assertLogsLen(t, 1))
		cachedPrices, err := cachingFCD.GasPrices()
		require.NoError(t, err)
		assert.Equal(t, prices["ucosm"], cachedPrices["ucosm"])
	})

	t.Run("closure", func(t *testing.T) {
		gpe := NewClosureGasPriceEstimator(func() (map[string]sdk.DecCoin, error) {
			return map[string]sdk.DecCoin{
				"ucosm": sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("10")),
			}, nil
		})
		p, err := gpe.GasPrices()
		require.NoError(t, err)
		price, ok := p["ucosm"]
		require.True(t, ok)
		assert.Equal(t, "ucosm", price.Denom)
		assert.Equal(t, "10.000000000000000000", price.Amount.String())
	})

	t.Run("composed", func(t *testing.T) {
		gpeFCD := NewFCDGasPriceEstimator(newConfig(t, "https://does.not.exist:443/v1/txs/gas_prices"), 10*time.Second, lggr)
		cachingFCD := NewCachingGasPriceEstimator(gpeFCD, lggr)
		gpeFixed := NewFixedGasPriceEstimator(map[string]sdk.DecCoin{
			"ucosm": sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("10")),
		})
		gpe := NewMustGasPriceEstimator([]GasPricesEstimator{cachingFCD, gpeFixed}, lggr)
		t.Cleanup(assertLogsLen(t, 1))
		fixedPrices := gpe.GasPrices()
		ucosm, ok := fixedPrices["ucosm"]
		assert.True(t, ok)
		assert.Equal(t, "10.000000000000000000", ucosm.Amount.String())
		// If the url starts working, it should use that.
		const goodURL = "https://fcd.terra.dev:443/v1/txs/gas_prices"
		gpeFCD.cfg = newConfig(t, goodURL)
		fcdPrices := gpe.GasPrices()
		ucosm, ok = fcdPrices["ucosm"]
		assert.True(t, ok)
		assert.NotEqual(t, "10.000000000000000000", ucosm.Amount.String())
	})
}

type config struct {
	fcdURL url.URL
}

func newConfig(t *testing.T, u string) *config {
	parsed, err := url.Parse(u)
	require.NoError(t, err)
	return &config{*parsed}
}

func (c *config) FCDURL() url.URL {
	return c.fcdURL
}
