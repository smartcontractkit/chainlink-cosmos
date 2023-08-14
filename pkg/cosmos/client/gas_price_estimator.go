package client

import (
	"fmt"
	"math/big"

	"github.com/smartcontractkit/chainlink-relay/pkg/fee"
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
	lggr      logger.SugaredLogger
}

func NewFixedGasPriceEstimator(prices map[string]sdk.DecCoin, lggr logger.SugaredLogger) *FixedGasPriceEstimator {
	return &FixedGasPriceEstimator{gasPrices: prices, lggr: lggr}
}

func (gpe *FixedGasPriceEstimator) GasPrices() (map[string]sdk.DecCoin, error) {
	return gpe.gasPrices, nil
}

func (gpe *FixedGasPriceEstimator) GasPrice(coin string) sdk.DecCoin {
	return gpe.gasPrices[coin]
}

func (gpe *FixedGasPriceEstimator) SetGasPrice(coin string, price sdk.DecCoin) {
	gpe.gasPrices[coin] = price
}

// CalculateGasPrice calculates the minimum of the maximum input gas prices.
// Parameters:
// - maxGasPrice: max gas price set by the user or operator
// - defaultGasPrice: default gas price
// - maxGasPriceConfigured: max gas price originally configured for the node
func (gpe *FixedGasPriceEstimator) CalculateGasPrice(
	coin string,
	maxGasPrice,
	defaultGasPrice,
	maxGasPriceConfigured sdk.DecCoin,
) sdk.DecCoin {
	calculatedGasPrice := fee.CalculateFee(
		maxGasPrice.Amount.BigInt(),
		defaultGasPrice.Amount.BigInt(),
		maxGasPriceConfigured.Amount.BigInt(),
	)
	return sdk.NewDecCoinFromDec(coin, sdk.NewDecFromBigIntWithPrec(calculatedGasPrice, sdk.Precision))
}

// BumpGasPrice calculates a new gas price by bumping the current gas price by a percentage.
// Parameters:
// - currentGasPrice: current gas price (could be after bumping)
// - originalGasPrice: original gas price (before bumping)
// - maxGasPrice: max gas price
// - maxBumpPrice: max gas price that can be bumped to
// - bumpMin: min gas price that can be bumped by
// - bumpPercent: percentage to bump by
func (gpe *FixedGasPriceEstimator) CalculateBumpGasPrice(
	coin string,
	currentGasPrice,
	originalGasPrice,
	maxGasPrice,
	maxBumpPrice,
	bumpMin sdk.DecCoin,
	bumpPercent uint16,
) (sdk.DecCoin, error) {
	bumpedGasPrice, err := fee.CalculateBumpedFee(
		gpe.lggr,
		currentGasPrice.Amount.BigInt(),
		originalGasPrice.Amount.BigInt(),
		maxGasPrice.Amount.BigInt(),
		maxBumpPrice.Amount.BigInt(),
		bumpMin.Amount.BigInt(),
		bumpPercent,
		FormatGasPrice,
	)
	if err != nil {
		return sdk.DecCoin{}, err
	}
	return sdk.NewDecCoinFromDec(coin, sdk.NewDecFromBigIntWithPrec(bumpedGasPrice, 18)), nil
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

func FormatGasPrice(gasPrice *big.Int) string {
	return sdk.NewDecFromBigInt(gasPrice).String()
}
