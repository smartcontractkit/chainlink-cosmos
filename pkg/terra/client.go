package terra

import (
	"context"
	"net/http"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/smartcontractkit/terra.go/msg"
	abci "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

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

type ReaderWriter interface {
	Writer
	Reader
}

type Reader interface {
	QueryABCI(path string, params ABCIQueryParams) (abci.ResponseQuery, error)
	GasPrice() msg.DecCoin
	Account(address sdk.AccAddress) (authtypes.AccountI, error)
	TxSearch(query string) (*ctypes.ResultTxSearch, error)
	Block(height *int64) (*ctypes.ResultBlock, error)
}

type Writer interface {
	// Assumes all msgs are for the same from address.
	// We may want to support multiple from addresses + signers if a use case arises.
	SignAndBroadcast(msgs []msg.Msg, accountNum uint64, sequence uint64, gasPrice sdk.DecCoin, signer key.PrivKey, mode txtypes.BroadcastMode) (*sdk.TxResponse, error)
}

var _ ReaderWriter = (*Client)(nil)

const (
	DefaultTimeout = 5 * time.Second
)

type Client struct {
	codec *codec.LegacyAmino

	fallbackGasPrice   sdk.Dec
	gasLimitMultiplier sdk.Dec
	fcdURL    string
	cosmosURL string
	chainID   string
	clientCtx          cosmosclient.Context

	// Timeout for node interactions
	timeout time.Duration

	log Logger
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

	if spec.Timeout == time.Duration(0) {
		spec.Timeout = DefaultTimeout
	}

	return &Client{
		codec:              codec.NewLegacyAmino(),
		chainID:            spec.ChainID,
		clientCtx:          clientCtx,
		cosmosURL:          spec.CosmosURL,
		timeout:            spec.Timeout,
		fcdURL:             spec.FcdURL,
		fallbackGasPrice:   fallbackGasPrice,
		gasLimitMultiplier: gasLimitMultiplier,
		log:                lggr,
	}, nil
}

func (c *Client) Account(addr sdk.AccAddress) (authtypes.AccountI, error) {
	lcd := client.NewLCDClient(c.cosmosURL, c.chainID, msg.NewDecCoinFromDec("uluna", c.fallbackGasPrice), c.gasLimitMultiplier, nil, c.timeout)
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	a, err := lcd.LoadAccount(ctx, addr)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (c *Client) GasPrice() msg.DecCoin {
	var fallback = msg.NewDecCoinFromDec("uluna", c.fallbackGasPrice)
	url := fmt.Sprintf("%s%s", c.fcdURL, "/v1/txs/gas_prices")
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.log.Errorf("error querying %s, err %v", url, err)
		return fallback
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.log.Errorf("error reading body, err %v", url, err)
		return fallback
	}
	defer resp.Body.Close()
	var prices struct {
		Uluna string `json:"uluna"`
	}
	if err := json.Unmarshal(b, &prices); err != nil {
		c.log.Errorf("error unmarshalling, err %v", url, err)
		return fallback
	}
	p, err := msg.NewDecFromStr(prices.Uluna)
	if err != nil {
		c.log.Errorf("error parsing, err %v", url, err)
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

func (c *Client) QueryABCI(path string, params ABCIQueryParams) (abci.ResponseQuery, error) {
	var resp abci.ResponseQuery
	data, err := c.codec.MarshalJSON(params)
	if err != nil {
		return resp, err
	}
	// TODO: unfortunately the cosmos client doesn't let you pass in a ctx
	// here for timing out
	resp, err = c.clientCtx.QueryABCI(abci.RequestQuery{
		Data:   data,
		Path:   path,
		Height: 0,
		Prove:  false,
	})
	return resp, err
}

func (c *Client) TxSearch(query string) (*ctypes.ResultTxSearch, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.clientCtx.Client.TxSearch(ctx, query, false, nil, nil, "desc")
}

func (c *Client) Block(height *int64) (*ctypes.ResultBlock, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.clientCtx.Client.Block(ctx, height)
}

func (c *Client) SignAndBroadcast(msgs []msg.Msg, account uint64, sequence uint64, gasPrice sdk.DecCoin, signer key.PrivKey, mode txtypes.BroadcastMode) (*sdk.TxResponse, error) {
	lcd := client.NewLCDClient(c.cosmosURL, c.chainID, gasPrice, c.gasLimitMultiplier, signer, c.timeout)
	// TODO: may want a different timeout for simulation
	// tempted to just remove LCD...
	simCtx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	// Don't set the gas limit and it will automatically estimate
	// the gas limit by simulating.
	txBuilder, err := lcd.CreateAndSignTx(simCtx, client.CreateTxOptions{
		Msgs: msgs,
		// Quirk of lcd, you have to specify the account number if you want to specify the sequence
		AccountNumber: account,
		Sequence:      sequence,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error in Transmit.NewTxBuilder")
	}
	broadcastCtx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return lcd.Broadcast(broadcastCtx, txBuilder, mode)
}
