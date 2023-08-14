package client

import (
	"errors"
	"fmt"
	"testing"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"

	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGasPriceEstimators(t *testing.T) {
	lggr, logs := logger.TestObserved(t, zap.WarnLevel)
	sugaredLggr := logger.Sugared(lggr)
	assertLogsLen := func(t *testing.T, l int) func() {
		return func() { assert.Len(t, logs.TakeAll(), l) }
	}

	t.Run("fixed", func(t *testing.T) {
		gpeFixed := NewFixedGasPriceEstimator(map[string]sdk.DecCoin{
			"ucosm": sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("10")),
		}, sugaredLggr)
		p, err := gpeFixed.GasPrices()
		require.NoError(t, err)
		price, ok := p["ucosm"]
		require.True(t, ok)
		assert.Equal(t, "ucosm", price.Denom)
		assert.Equal(t, "10.000000000000000000", price.Amount.String())
	})

	t.Run("caching", func(t *testing.T) {
		responses := []sdk.DecCoin{
			sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("10")),
		}
		gpe := NewClosureGasPriceEstimator(func() (map[string]sdk.DecCoin, error) {
			if len(responses) == 0 {
				return nil, errors.New("no more prices")
			}
			var price sdk.DecCoin
			price, responses = responses[0], responses[1:]
			return map[string]sdk.DecCoin{
				"ucosm": price,
			}, nil
		})
		cachedGpe := NewCachingGasPriceEstimator(gpe, lggr)

		// Fill cache
		prices, err := cachedGpe.GasPrices()
		require.NoError(t, err)

		// Use cache, no more prices returned from estimator
		t.Cleanup(assertLogsLen(t, 1))
		cachedPrices, err := cachedGpe.GasPrices()
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
		responses := []sdk.DecCoin{}
		closureGpe := NewClosureGasPriceEstimator(func() (map[string]sdk.DecCoin, error) {
			if len(responses) == 0 {
				return nil, errors.New("no more prices")
			}
			var price sdk.DecCoin
			price, responses = responses[0], responses[1:]
			return map[string]sdk.DecCoin{
				"ucosm": price,
			}, nil
		})
		cachingGpe := NewCachingGasPriceEstimator(closureGpe, lggr)
		gpeFixed := NewFixedGasPriceEstimator(map[string]sdk.DecCoin{
			"ucosm": sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("10")),
		}, sugaredLggr)
		gpe := NewMustGasPriceEstimator([]GasPricesEstimator{cachingGpe, gpeFixed}, lggr)
		t.Cleanup(assertLogsLen(t, 1))
		fixedPrices := gpe.GasPrices()
		ucosm, ok := fixedPrices["ucosm"]
		assert.True(t, ok)
		assert.Equal(t, "10.000000000000000000", ucosm.Amount.String())
		// If the url starts working, it should use that.
		responses = append(responses, sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("9")))
		gpePrices := gpe.GasPrices()
		ucosm, ok = gpePrices["ucosm"]
		assert.True(t, ok)
		assert.NotEqual(t, "10.000000000000000000", ucosm.Amount.String())
	})
}

func TestFixedPriceGasEstimator(t *testing.T) {
	lggr := logger.Sugared(logger.Test(t))

	t.Run("calculate gas price", func(t *testing.T) {
		gpeFixed := NewFixedGasPriceEstimator(map[string]sdk.DecCoin{}, lggr)
		tests := []struct {
			name            string
			defaultGasPrice sdk.DecCoin
			maxGasPrice     sdk.DecCoin
			maxGasPriceNode sdk.DecCoin
			want            sdk.DecCoin
		}{
			{name: "Returns default price when maxGasPrice and maxGasPriceNode are greater",
				defaultGasPrice: sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.001")),
				maxGasPrice:     sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.0011")),
				maxGasPriceNode: sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.0012")),
				want:            sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.001")),
			},
			{name: "Returns maxGasPrice when maxGasPrice is lower than defaultGasPrice and maxGasPriceNode",
				defaultGasPrice: sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.01")),
				maxGasPrice:     sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.009")),
				maxGasPriceNode: sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.012")),
				want:            sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.009")),
			},
			{name: "Returns maxGasPriceNode when maxGasPriceNode is lower than defaultGasPrice and maxGasPrice",
				defaultGasPrice: sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.010")),
				maxGasPrice:     sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.011")),
				maxGasPriceNode: sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.008")),
				want:            sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.008")),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := gpeFixed.CalculateGasPrice("ucosm", tt.maxGasPrice, tt.defaultGasPrice, tt.maxGasPriceNode)
				fmt.Println(tt.maxGasPrice.Amount)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("bump gas price", func(t *testing.T) {
		tests := []struct {
			name             string
			currentGasPrice  sdk.DecCoin
			originalGasPrice sdk.DecCoin
			maxGasPrice      sdk.DecCoin
			maxBumpPrice     sdk.DecCoin
			minBumpPrice     sdk.DecCoin
			bumpPercent      uint16
			want             sdk.DecCoin
		}{
			{
				name:             "Bump the gas price by minimum as bumpPercent is less than bumpMin",
				currentGasPrice:  sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.001")),
				originalGasPrice: sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.001")),
				maxGasPrice:      sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.02")),
				maxBumpPrice:     sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.3")),
				minBumpPrice:     sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.0005")),
				bumpPercent:      30,
				want:             sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.0015")),
			},
			{
				name:             "Bump the gas price by 30% as bumpPercent is greater than bumpMin",
				currentGasPrice:  sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.0010")),
				originalGasPrice: sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.001")),
				maxGasPrice:      sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.02")),
				maxBumpPrice:     sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.3")),
				minBumpPrice:     sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.00005")),
				bumpPercent:      30,
				want:             sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.0013")),
			},
		}

		gpeFixed := NewFixedGasPriceEstimator(map[string]sdk.DecCoin{
			"ucosm": sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.001")),
		}, lggr)

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bumpedGasPrice, err := gpeFixed.CalculateBumpGasPrice("ucosm", tt.currentGasPrice, tt.originalGasPrice, tt.maxGasPrice, tt.maxBumpPrice, tt.minBumpPrice, tt.bumpPercent)
				require.NoError(t, err)
				gpeFixed.SetGasPrice("ucosm", bumpedGasPrice)
				assert.Equal(t, tt.want, gpeFixed.GasPrice("ucosm"))
			})
		}
	})
}
