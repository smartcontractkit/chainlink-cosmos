package terra

import (
	"context"
	tmtypes "github.com/tendermint/tendermint/types"
	"net/http"

	"encoding/json"
	"fmt"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/pkg/errors"
	"github.com/terra-money/core/app"
	"io/ioutil"

	"time"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/terra-money/terra.go/client"
	"github.com/terra-money/terra.go/key"
	"github.com/terra-money/terra.go/msg"
)

type Client struct {
	close chan struct{}
	codec *codec.LegacyAmino

	fallbackGasPrice   msg.Dec
	gasLimitMultiplier msg.Dec
	fcdhttpURL         string
	cosmosRPC          string
	chainID            string
	clientCtx          cosmosclient.Context
	httpClient         *http.Client

	Height uint64
	Log    Logger
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
		close:              make(chan struct{}),
		codec:              codec.NewLegacyAmino(),
		chainID:            spec.ChainID,
		clientCtx:          clientCtx,
		cosmosRPC:          spec.CosmosURL,
		httpClient:         &http.Client{Timeout: spec.HTTPTimeout},
		fcdhttpURL:         spec.FCDNodeEndpointURL,
		fallbackGasPrice:   fallbackGasPrice,
		gasLimitMultiplier: gasLimitMultiplier,
		Log:                lggr,
	}, nil
}

func (c *Client) Start() error {
	// Note starts the websocket and head tracker
	if err := c.clientCtx.Client.Start(); err != nil {
		return err
	}
	blocks, err := c.clientCtx.Client.Subscribe(context.TODO(), "head-tracker", "tm.event='NewBlock'")
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case block := <-blocks:
				b, ok := block.Data.(tmtypes.EventDataNewBlock)
				fmt.Println("got a block", b, ok)
				if !ok {
					c.Log.Errorf("[head-tracker] did not get block, got %T", block)
					continue
				}
				c.Log.Infof("[head-tracker] Block height %d", b.Block.Height)
				c.Height = uint64(b.Block.Height)
			case <-c.close:
				return
			}
		}
	}()
	c.Log.Infof("[head-tracker] Subscription started")
	return nil
}

func (c *Client) Close() error {
	if err := c.clientCtx.Client.Unsubscribe(context.TODO(), "head-tracker", "tm.event='NewBlock'"); err != nil {
		return err
	}
	// trigger close channel to trigger stop related services
	close(c.close)
	c.Log.Infof("Closing websocket connection to %s", c.clientCtx.Client.String())
	return nil
}

func (c *Client) LCD(gasPrice msg.DecCoin, gasAdjustment msg.Dec, signer key.PrivKey, timeout time.Duration) *client.LCDClient {
	return client.NewLCDClient(c.cosmosRPC, c.chainID, gasPrice, gasAdjustment, signer, timeout)
}

func (c *Client) GasPrice() msg.DecCoin {
	var fallback = msg.NewDecCoinFromDec("uluna", c.fallbackGasPrice)
	url := fmt.Sprintf("%s%s", c.fcdhttpURL, "/v1/txs/gas_prices")
	resp, err := c.httpClient.Get(url)
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
	ContractID string
	Msg        []byte
}

func NewAbciQueryParams(contractID string, msg []byte) ABCIQueryParams {
	return ABCIQueryParams{contractID, msg}
}
