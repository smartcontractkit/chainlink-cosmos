package terra

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/smartcontractkit/terra.go/msg"
	abci "github.com/tendermint/tendermint/abci/types"
	"net/http"

	"encoding/json"
	"fmt"
	"io/ioutil"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/pkg/errors"
	"github.com/terra-money/core/app"

	"time"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/smartcontractkit/terra.go/client"
	"github.com/smartcontractkit/terra.go/key"
)

type TerraReaderWriter interface {
	TerraWriter
	TerraReader
}

type TerraReader interface {
	QueryABCI(path string, params ABCIQueryParams) (abci.ResponseQuery, error)
	GasPrice() msg.DecCoin
	SequenceNumber(address sdk.AccAddress) (uint64, error)
}

type TerraWriter interface {
	SignAndBroadcast(msg msg.Msg, sequence uint64, gasPrice sdk.DecCoin, signer key.PrivKey) (*sdk.TxResponse, error)
}

type Client struct {
	codec *codec.LegacyAmino

	fallbackGasPrice   sdk.Dec
	gasLimitMultiplier sdk.Dec
	fcdhttpURL         string
	cosmosRPC          string
	chainID            string
	clientCtx          cosmosclient.Context
	httpTimeout        time.Duration

	Log Logger
}

func NewClient(spec OCR2Spec, lggr Logger) (*Client, error) {
	fallbackGasPrice, err := sdk.NewDecFromStr(spec.FallbackGasPrice)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid fallback gas price %v", spec.FallbackGasPrice)
	}
	gasLimitMultiplier, err := sdk.NewDecFromStr(spec.GasLimitMultiplier)
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

func (c *Client) SequenceNumber(addr sdk.AccAddress) (uint64, error) {
	lcd := client.NewLCDClient(c.cosmosRPC, c.chainID, msg.NewDecCoinFromDec("uluna", c.fallbackGasPrice), c.gasLimitMultiplier, nil, c.httpTimeout)
	a, err := lcd.LoadAccount(context.TODO(), addr)
	if err != nil {
		return 0, err
	}
	return a.GetSequence(), nil
}

func (c *Client) SignAndBroadcast(m msg.Msg, sequence uint64, gasPrice sdk.DecCoin, signer key.PrivKey) (*sdk.TxResponse, error) {
	lcd := client.NewLCDClient(c.cosmosRPC, c.chainID, gasPrice, c.gasLimitMultiplier, signer, c.httpTimeout)
	ctx, cancel := context.WithTimeout(context.Background(), c.httpTimeout)
	defer cancel()
	// Don't set the gas limit and it will automatically estimate
	// the gas limit by simulating.
	txBuilder, err := lcd.CreateAndSignTx(ctx, client.CreateTxOptions{
		Msgs:     []msg.Msg{m},
		Sequence: sequence,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error in Transmit.NewTxBuilder")
	}
	return lcd.Broadcast(ctx, txBuilder, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
}

//func (c *Client) LCD(gasPrice msg.DecCoin, gasAdjustment msg.Dec, signer key.PrivKey, timeout time.Duration) *client.LCDClient {
//}
//
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

func (c *Client) QueryABCI(path string, params ABCIQueryParams) (abci.ResponseQuery, error) {
	var resp abci.ResponseQuery
	data, err := c.codec.MarshalJSON(params)
	if err != nil {
		return resp, err
	}
	resp, err = c.clientCtx.QueryABCI(abci.RequestQuery{
		Data:   data,
		Path:   path,
		Height: 0,
		Prove:  false,
	})
	return resp, err
}

type ABCIQueryParams struct {
	ContractAddress string
	Msg             []byte
}

func NewAbciQueryParams(contractAddress string, msg []byte) ABCIQueryParams {
	return ABCIQueryParams{contractAddress, msg}
}
