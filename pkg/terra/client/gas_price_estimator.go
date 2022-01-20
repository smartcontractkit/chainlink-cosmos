package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type GasPricesEstimator interface {
	GasPrices() map[string]sdk.DecCoin
}

var _ GasPricesEstimator = (*FixedGasPriceEstimator)(nil)

type FixedGasPriceEstimator struct {
	gasPrices map[string]sdk.DecCoin
}

func NewFixedGasPriceEstimator(prices map[string]sdk.DecCoin) *FixedGasPriceEstimator {
	return &FixedGasPriceEstimator{gasPrices: prices}
}

func (gpe *FixedGasPriceEstimator) GasPrices() map[string]sdk.DecCoin {
	return gpe.gasPrices
}

type FCDGasPriceEstimator struct {
	fcdURL url.URL
	prices map[string]sdk.DecCoin
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
	initialGasPrices, err := gpe.request()
	if err != nil {
		return nil, err
	}
	gpe.prices = initialGasPrices
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
		name, value := v.Type().Field(i).Name, v.Field(i).String()
		amount, err := sdk.NewDecFromStr(value)
		if err != nil {
			return nil, err
		}
		results[name] = sdk.NewDecCoinFromDec(name, amount)
	}
	return results, nil
}

func (gpe *FCDGasPriceEstimator) GasPrices() map[string]sdk.DecCoin {
	latestGasPrice, err := gpe.request()
	if err != nil {
		gpe.lggr.Warnf("unable get latest prices, using last cached value", "cached value", gpe.prices, "err", err)
		return gpe.prices
	}
	return latestGasPrice
}
