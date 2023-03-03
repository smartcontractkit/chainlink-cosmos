package client

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"go.uber.org/multierr"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type GasPricesEstimator interface {
	GasPrices() (map[string]sdk.DecCoin, error)
}

var _ GasPricesEstimator = (*FixedGasPriceEstimator)(nil)

type FixedGasPriceEstimator struct {
	gasPrices map[string]sdk.DecCoin
}

func NewFixedGasPriceEstimator(prices map[string]sdk.DecCoin) *FixedGasPriceEstimator {
	return &FixedGasPriceEstimator{gasPrices: prices}
}

func (gpe *FixedGasPriceEstimator) GasPrices() (map[string]sdk.DecCoin, error) {
	return gpe.gasPrices, nil
}

// Useful for hot reloads of configured prices
type ClosureGasPriceEstimator struct {
	gasPrices func() (map[string]sdk.DecCoin, error)
}

func NewClosureGasPriceEstimator(prices func() (map[string]sdk.DecCoin, error)) *ClosureGasPriceEstimator {
	return &ClosureGasPriceEstimator{gasPrices: prices}
}

func (gpe *ClosureGasPriceEstimator) GasPrices() (map[string]sdk.DecCoin, error) {
	return gpe.gasPrices()
}

var _ GasPricesEstimator = (*CachingGasPriceEstimator)(nil)

type CachingGasPriceEstimator struct {
	lastPrices map[string]sdk.DecCoin
	estimator  GasPricesEstimator
	lggr       logger.Logger
}

func NewCachingGasPriceEstimator(estimator GasPricesEstimator, lggr logger.Logger) *CachingGasPriceEstimator {
	return &CachingGasPriceEstimator{estimator: estimator, lggr: lggr}
}

func (gpe *CachingGasPriceEstimator) GasPrices() (map[string]sdk.DecCoin, error) {
	latestPrices, err := gpe.estimator.GasPrices()
	if err != nil {
		if gpe.lastPrices == nil {
			return nil, fmt.Errorf("unable to get gas prices and cache is empty: %w", err)
		}
		gpe.lggr.Warnf("error %v getting latest prices, using cached value %v", err, gpe.lastPrices)
		return gpe.lastPrices, nil
	}
	gpe.lastPrices = latestPrices
	return latestPrices, nil
}

type ComposedGasPriceEstimator struct {
	estimators []GasPricesEstimator
	lggr       logger.Logger
}

func NewMustGasPriceEstimator(estimators []GasPricesEstimator, lggr logger.Logger) *ComposedGasPriceEstimator {
	return &ComposedGasPriceEstimator{estimators: estimators, lggr: lggr}
}

func (gpe *ComposedGasPriceEstimator) GasPrices() map[string]sdk.DecCoin {
	// Try each estimator in order
	var finalError error
	for _, estimator := range gpe.estimators {
		latestPrices, err := estimator.GasPrices()
		if err != nil {
			finalError = multierr.Combine(finalError, err)
			gpe.lggr.Warnf("error using estimator, trying next one, err %v", err)
			continue
		}
		return latestPrices
	}
	panic(fmt.Sprintf("no estimator succeeded errs %v", finalError))
}
