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

type FCDGasPriceEstimator struct {
	fcdURL url.URL
	client http.Client
	lggr   Logger
}

func NewFCDGasPriceEstimator(fcdURLRaw string, requestTimeout time.Duration, lggr Logger) (*FCDGasPriceEstimator, error) {
	// Sanity check the URL works and populate the cached value
	fcdURL, err := url.Parse(fcdURLRaw)
	if err != nil {
		return nil, err
	}
	client := http.Client{Timeout: requestTimeout}
	gpe := FCDGasPriceEstimator{fcdURL: *fcdURL, client: client, lggr: lggr}
	return &gpe, nil
}

type prices struct {
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
	req, _ := http.NewRequest(http.MethodGet, gpe.fcdURL.String(), nil)
	resp, err := gpe.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		gpe.lggr.Errorf("error reading body, err %v", gpe.fcdURL.RequestURI(), err)
		return nil, err
	}
	var prices prices
	if err := json.Unmarshal(b, &prices); err != nil {
		gpe.lggr.Errorf("error unmarshalling, err %v", gpe.fcdURL.RequestURI(), err)
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
