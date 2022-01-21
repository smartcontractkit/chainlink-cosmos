package client

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGasPriceEstimators(t *testing.T) {
	lggr := new(mocks.Logger)
	lggr.Test(t)
	t.Run("fixed", func(t *testing.T) {
		gpeFixed := NewFixedGasPriceEstimator(map[string]sdk.DecCoin{
			"uluna": sdk.NewDecCoinFromDec("uluna", sdk.MustNewDecFromStr("10")),
		})
		p, err := gpeFixed.GasPrices()
		require.NoError(t, err)
		price, ok := p["uluna"]
		require.True(t, ok)
		assert.Equal(t, "uluna", price.Denom)
		assert.Equal(t, "10.000000000000000000", price.Amount.String())
	})

	t.Run("fcd", func(t *testing.T) {
		// Note this test runs in CI against a real api, we do want to know if this API changes or becomes slow
		gpeFCD, err := NewFCDGasPriceEstimator("https://fcd.terra.dev:443/v1/txs/gas_prices", 10*time.Second, lggr)
		require.NoError(t, err)
		p, err := gpeFCD.GasPrices()
		require.NoError(t, err)
		for _, price := range []string{
			"uluna",
			"usdr",
			"ukrw",
			"umnt",
			"ueur",
			"ucny",
			"ujpy",
			"ugbp",
			"uinr",
			"ucad",
			"uchf",
			"uaud",
			"usgd",
			"uthb",
			"usek",
			"unok",
			"udkk",
			"uidr",
			"uphp",
			"uhkd",
		} {
			_, ok := p[price]
			assert.True(t, ok)
		}
	})

	t.Run("caching", func(t *testing.T) {
		gpeFCD, err := NewFCDGasPriceEstimator("https://fcd.terra.dev:443/v1/txs/gas_prices", 10*time.Second, lggr)
		require.NoError(t, err)
		cachingFCD := NewCachingGasPriceEstimator(gpeFCD, lggr)

		// Fill cache
		prices, err := cachingFCD.GasPrices()
		require.NoError(t, err)

		// Use cache
		badURL, err := url.Parse("https://does.not.exist:443/v1/txs/gas_prices")
		require.NoError(t, err)
		gpeFCD.fcdURL = *badURL
		lggr.On("Warnf", mock.Anything, mock.Anything, mock.Anything).Once()
		cachedPrices, err := cachingFCD.GasPrices()
		require.NoError(t, err)
		assert.Equal(t, prices["uluna"], cachedPrices["uluna"])
	})

	t.Run("closure", func(t *testing.T) {
		gpe := NewClosureGasPriceEstimator(func() (map[string]sdk.DecCoin, error) {
			return map[string]sdk.DecCoin{
				"uluna": sdk.NewDecCoinFromDec("uluna", sdk.MustNewDecFromStr("10")),
			}, nil
		})
		p, err := gpe.GasPrices()
		require.NoError(t, err)
		price, ok := p["uluna"]
		require.True(t, ok)
		assert.Equal(t, "uluna", price.Denom)
		assert.Equal(t, "10.000000000000000000", price.Amount.String())
	})

	t.Run("composed", func(t *testing.T) {
		gpeFCD, err := NewFCDGasPriceEstimator("https://does.not.exist:443/v1/txs/gas_prices", 10*time.Second, lggr)
		require.NoError(t, err)
		cachingFCD := NewCachingGasPriceEstimator(gpeFCD, lggr)
		gpeFixed := NewFixedGasPriceEstimator(map[string]sdk.DecCoin{
			"uluna": sdk.NewDecCoinFromDec("uluna", sdk.MustNewDecFromStr("10")),
		})
		gpe := NewMustGasPriceEstimator([]GasPricesEstimator{cachingFCD, gpeFixed}, lggr)
		lggr.On("Warnf", mock.Anything, mock.Anything, mock.Anything).Twice()
		fixedPrices := gpe.GasPrices()
		uluna, ok := fixedPrices["uluna"]
		assert.True(t, ok)
		assert.Equal(t, "10.000000000000000000", uluna.Amount.String())
		// If the url starts working, it should use that.
		goodURL, err := url.Parse("https://fcd.terra.dev:443/v1/txs/gas_prices")
		require.NoError(t, err)
		gpeFCD.fcdURL = *goodURL
		fcdPrices := gpe.GasPrices()
		uluna, ok = fcdPrices["uluna"]
		assert.True(t, ok)
		assert.NotEqual(t, "10.000000000000000000", uluna.Amount.String())
	})
}
