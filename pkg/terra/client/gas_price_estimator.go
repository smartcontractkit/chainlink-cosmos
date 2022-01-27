package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
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

var _ GasPricesEstimator = (*FCDGasPriceEstimator)(nil)

// FCDGasPriceEstimator is a GasPricesEstimator which fetches from the latest configured fcd url.
type FCDGasPriceEstimator struct {
	cfg    Config
	client http.Client
	lggr   Logger
}

// Config is a subset of pkg/terra.Config, which cannot be imported here.
type Config interface{ FCDURL() url.URL }

func NewFCDGasPriceEstimator(cfg Config, requestTimeout time.Duration, lggr Logger) *FCDGasPriceEstimator {
	client := http.Client{Timeout: requestTimeout}
	gpe := FCDGasPriceEstimator{cfg: cfg, client: client, lggr: lggr}
	return &gpe
}

type pricesFCD struct {
	Uluna string `json:"uluna"`
	Usdr  string `json:"usdr"`
	Uusd  string `json:"uusd"`
	Ukrw  string `json:"ukrw"`
	Umnt  string `json:"umnt"`
	Ueur  string `json:"ueur"`
	Ucny  string `json:"ucny"`
	Ujpy  string `json:"ujpy"`
	Ugbp  string `json:"ugbp"`
	Uinr  string `json:"uinr"`
	Ucad  string `json:"ucad"`
	Uchf  string `json:"uchf"`
	Uaud  string `json:"uaud"`
	Usgd  string `json:"usgd"`
	Uthb  string `json:"uthb"`
	Usek  string `json:"usek"`
	Unok  string `json:"unok"`
	Udkk  string `json:"udkk"`
	Uidr  string `json:"uidr"`
	Uphp  string `json:"uphp"`
	Uhkd  string `json:"uhkd"`
}

func (gpe *FCDGasPriceEstimator) request() (map[string]sdk.DecCoin, error) {
	fcdURL := gpe.cfg.FCDURL()
	if fcdURL == (url.URL{}) {
		return nil, errors.New("fcd url missing from chain config")
	}
	req, err := http.NewRequest(http.MethodGet, fcdURL.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := gpe.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		gpe.lggr.Errorf("error reading body from %s, err %v", req.URL.RequestURI(), err)
		return nil, err
	}
	var prices pricesFCD
	if err := json.Unmarshal(b, &prices); err != nil {
		gpe.lggr.Errorf("error unmarshalling from %s, err %v", req.URL.RequestURI(), err)
		return nil, err
	}
	results := make(map[string]sdk.DecCoin)
	v := reflect.ValueOf(prices)
	for i := 0; i < v.NumField(); i++ {
		name, value := strings.ToLower(v.Type().Field(i).Name), v.Field(i).String()
		amount, err := sdk.NewDecFromStr(value)
		if err != nil {
			return nil, err
		}
		results[name] = sdk.NewDecCoinFromDec(name, amount)
	}
	return results, nil
}

func (gpe *FCDGasPriceEstimator) GasPrices() (map[string]sdk.DecCoin, error) {
	return gpe.request()
}

var _ GasPricesEstimator = (*CachingGasPriceEstimator)(nil)

type CachingGasPriceEstimator struct {
	lastPrices map[string]sdk.DecCoin
	estimator  GasPricesEstimator
	lggr       Logger
}

func NewCachingGasPriceEstimator(estimator GasPricesEstimator, lggr Logger) *CachingGasPriceEstimator {
	return &CachingGasPriceEstimator{estimator: estimator, lggr: lggr}
}

func (gpe *CachingGasPriceEstimator) GasPrices() (map[string]sdk.DecCoin, error) {
	latestPrices, err := gpe.estimator.GasPrices()
	if err != nil {
		if gpe.lastPrices == nil {
			return nil, errors.Errorf("unable to get gas prices and cache is empty, err %v", err)
		}
		gpe.lggr.Warnf("error %v getting latest prices, using cached value %v", err, gpe.lastPrices)
		return gpe.lastPrices, nil
	}
	gpe.lastPrices = latestPrices
	return latestPrices, nil
}

type ComposedGasPriceEstimator struct {
	estimators []GasPricesEstimator
	lggr       Logger
}

func NewMustGasPriceEstimator(estimators []GasPricesEstimator, lggr Logger) *ComposedGasPriceEstimator {
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
