package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type GasPriceEstimator interface {
	MustGasPrice(denom []string) []sdk.DecCoin
}

var _ GasPriceEstimator = (*FixedGasPriceEstimator)(nil)

type FixedGasPriceEstimator struct {
	gasPrice sdk.DecCoin
}

func NewFixedGasPriceEstimator(price sdk.DecCoin) *FixedGasPriceEstimator {
	return &FixedGasPriceEstimator{gasPrice: price}
}

func (gpe *FixedGasPriceEstimator) MustGasPrice(denoms []string) []sdk.DecCoin {
	return []sdk.DecCoin{gpe.gasPrice}
}

type FCDGasPriceEstimator struct {
	fcdURL url.URL
	prices prices
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
	initialGasPrice, err := gpe.request()
	if err != nil {
		return nil, err
	}
	gpe.prices = *initialGasPrice
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

func (gpe *FCDGasPriceEstimator) request() (*prices, error) {
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
	return &prices, nil
}

// Will skip invalid denominations
func (gpe *FCDGasPriceEstimator) MustGasPrice(denoms []string) []sdk.DecCoin {
	var gasPrices = gpe.prices
	latestGasPrice, err := gpe.request()
	if err != nil {
		gasPrices = *latestGasPrice
	}
	var res []sdk.DecCoin
	for _, denom := range denoms {
		if err := sdk.ValidateDenom(denom); err != nil {
			gpe.lggr.Errorf("invalid denom %v, skipping", err)
			continue
		}
		field := reflect.ValueOf(gasPrices).FieldByName(strings.Title(denom))
		if !field.IsValid() {
			// Would mean there is a mismatch between the api and the sdk support denoms
			// Should never happen because of our initial query
			gpe.lggr.Errorf("unexpected error, mismatch between denoms and fcd api", err)
			continue
		}
		gasPriceAmount, err := sdk.NewDecFromStr(field.String())
		if err != nil {
			gpe.lggr.Errorf("unexpected error, unable to parse gas price", err)
			continue
		}
		res = append(res, sdk.NewDecCoinFromDec(denom, gasPriceAmount))
	}
	return res
}
