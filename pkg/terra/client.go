package terra

import (
	"context"
	"net/http"

	"encoding/json"
	"fmt"
	"io/ioutil"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/pkg/errors"
	"github.com/terra-money/core/app"

	"time"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/terra-money/terra.go/client"
	"github.com/terra-money/terra.go/key"
	"github.com/terra-money/terra.go/msg"
)

type Client struct {
	codec *codec.LegacyAmino

	fallbackGasPrice   msg.Dec
	gasLimitMultiplier msg.Dec
	fcdhttpURL         string
	cosmosRPC          string
	chainID            string
	clientCtx          cosmosclient.Context
	httpTimeout        time.Duration

	Log Logger
}

func NewClient(spec OCR2Spec, lggr Logger) (*Client, error) {
	fallbackGasPrice, err := msg.NewDecFromStr(spec.FallbackGasPrice)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid fallback gas price %v", spec.FallbackGasPrice)
	}
	gasLimitMultiplier, err := msg.NewDecFromStr(spec.GasLimitMultiplier)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid gas limit multiplier %v", spec.GasLimitMultiplier)
	}
	tmClient, err := cosmosclient.NewClientFromNode(spec.TendermintURL)
	if err != nil {
		return nil, err
	}
	clientCtx := cosmosclient.Context{}.
		WithClient(tmClient).
		WithChainID(spec.ChainID).
		WithTxConfig(app.MakeEncodingConfig().TxConfig)

	return &Client{
		codec:              codec.NewLegacyAmino(),
		chainID:            spec.ChainID,
		clientCtx:          clientCtx,
		cosmosRPC:          spec.CosmosURL,
		httpTimeout:        spec.HTTPTimeout,
		fcdhttpURL:         spec.FCDNodeEndpointURL,
		fallbackGasPrice:   fallbackGasPrice,
		gasLimitMultiplier: gasLimitMultiplier,
		Log:                lggr,
	}, nil
}

func (c *Client) LCD(gasPrice msg.DecCoin, gasAdjustment msg.Dec, signer key.PrivKey, timeout time.Duration) *client.LCDClient {
	return client.NewLCDClient(c.cosmosRPC, c.chainID, gasPrice, gasAdjustment, signer, timeout)
}

func (c *Client) GasPrice() msg.DecCoin {
	var fallback = msg.NewDecCoinFromDec("uluna", c.fallbackGasPrice)
	url := fmt.Sprintf("%s%s", c.fcdhttpURL, "/v1/txs/gas_prices")
	ctx, cancel := context.WithTimeout(context.Background(), c.httpTimeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Log.Errorf("error querying %s, err %v", url, err)
		return fallback
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.Log.Errorf("error reading body, err %v", url, err)
		return fallback
	}
	defer resp.Body.Close()
	var prices struct {
		Uluna string `json:"uluna"`
	}
	if err := json.Unmarshal(b, &prices); err != nil {
		c.Log.Errorf("error unmarshalling, err %v", url, err)
		return fallback
	}
	p, err := msg.NewDecFromStr(prices.Uluna)
	if err != nil {
		c.Log.Errorf("error parsing, err %v", url, err)
		return fallback
	}
	return msg.NewDecCoinFromDec("uluna", p)
}

type ABCIQueryParams struct {
	ContractAddress string
	Msg             []byte
}

func NewAbciQueryParams(contractAddress string, msg []byte) ABCIQueryParams {
	return ABCIQueryParams{contractAddress, msg}
}
