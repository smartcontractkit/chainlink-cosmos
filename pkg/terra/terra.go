package terra

import (
	//"bytes"
	"context"
	tmtypes "github.com/tendermint/tendermint/types"
	"net/http"

	//cmtypes "github.com/cosmos/cosmos-sdk/types"
	//"encoding/hex"
	"encoding/json"
	"fmt"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	//txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/pkg/errors"
	"github.com/terra-money/core/app"
	"io/ioutil"

	"time"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/terra-money/terra.go/client"
	"github.com/terra-money/terra.go/key"
	"github.com/terra-money/terra.go/msg"
	//abci "github.com/tendermint/tendermint/abci/types"
	//"github.com/tidwall/gjson"
)

//type WsConn struct {
//	mu   *sync.Mutex
//	conn *websocket.Conn
//}
//
//type RpcRequest struct {
//	Jsonrpc string
//	Method  string
//	Params  []interface{}
//	ID      string
//}
//
//type RpcMessage struct {
//	Error string
//	Data  []byte
//}

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

	//ws        WsConn
	//wsStarted bool
	//subs      map[string]subscription
	//subUnsub  map[string]chan<- Events
	// TODO(connor): If we use http we don't need this queryCh
	//queryCh chan RpcMessage

	Height uint64
	Log    Logger
}

func NewClient(spec OCR2Spec, lggr Logger) (Client, error) {
	fallbackGasPrice, err := msg.NewDecFromStr(spec.FallbackGasPrice)
	if err != nil {
		return Client{}, errors.Wrapf(err, "invalid fallback gas price %v", spec.FallbackGasPrice)
	}
	gasLimitMultiplier, err := msg.NewDecFromStr(spec.GasLimitMultiplier)
	if err != nil {
		return Client{}, errors.Wrapf(err, "invalid gas limit multiplier %v", spec.GasLimitMultiplier)
	}
	tmClient, err := cosmosclient.NewClientFromNode(spec.TendermintRPC)
	if err != nil {
		return Client{}, err
	}
	clientCtx := cosmosclient.Context{}.
		WithClient(tmClient).
		WithChainID(spec.ChainID).
		WithTxConfig(app.MakeEncodingConfig().TxConfig)

	return Client{
		close:      make(chan struct{}),
		codec:      codec.NewLegacyAmino(),
		chainID:    spec.ChainID,
		clientCtx:  clientCtx,
		cosmosRPC:  spec.CosmosRPC,
		httpClient: &http.Client{Timeout: spec.HTTPTimeout},
		//httpURL:            spec.NodeEndpointHTTP,
		//wsURL:              spec.NodeEndpointWS,
		fcdhttpURL:         spec.FCDNodeEndpointHTTP,
		fallbackGasPrice:   fallbackGasPrice,
		gasLimitMultiplier: gasLimitMultiplier,
		//subs:               make(map[string]subscription),
		//subUnsub:           make(map[string]chan<- Events),
		//queryCh:            make(chan RpcMessage),
		Log: lggr,
	}, nil
}

func (c Client) Start() error {
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
				if !ok {
					c.Log.Errorf("[head-tracker] did not get block, got %T", block)
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

func (c Client) Close() error {
	if err := c.clientCtx.Client.Unsubscribe(context.TODO(), "head-tracker", "tm.event='NewBlock'"); err != nil {
		return err
	}
	// trigger close channel to trigger stop related services
	close(c.close)
	c.Log.Infof("Closing websocket connection to %s", c.clientCtx.Client.String())
	return nil
}

func (c Client) LCD(gasPrice msg.DecCoin, gasAdjustment msg.Dec, signer key.PrivKey, timeout time.Duration) *client.LCDClient {
	return client.NewLCDClient(c.cosmosRPC, c.chainID, gasPrice, gasAdjustment, signer, timeout)
}

// Always returns a gas price,
func (c Client) GasPrice() msg.DecCoin {
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

//type subscription struct {
//	Channel chan<- Events
//	Payload request
//}
//
//type request struct {
//	Jsonrpc string
//	Method  string
//	Params  []string
//	ID      string
//}
/*
func (c *Client) HeadTracker() error {
	fq := []string{"tm.event='NewBlock'"}

	// create new block subscription
	channel := make(chan Events)
	if err := c.subscribe(context.TODO(), "head-tracker", fq, channel); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case height := <-channel:
				// update block number in the client
				c.Log.Infof("[head-tracker] Block height %d", height.Block)
				c.Height = height.Block
			case <-c.close:
				return
			}
		}
	}()

	c.Log.Infof("[head-tracker] Subscription started")
	return nil
}

// Subscribe to Terra events (address specifics)
func (c *Client) Subscribe(ctx context.Context, jobID string, address types.AccAddress, channel chan Events) error {
	fq := []string{fmt.Sprintf("tm.event='Tx' AND execute_contract.contract_address='%s'", address)}

	if err := c.subscribe(ctx, jobID, fq, channel); err != nil {
		return err
	}
	c.Log.Infof("[%s] Subscription created for %s", jobID, address)
	return nil
}

type Events struct {
	SubErr string   // error string if encountered during sub + unsub
	Block  uint64   // handle block number event
	Events []string // handle tx events
}
*/

// base level subscribe function
/*
func (c *Client) subscribe(ctx context.Context, jobID string, filterQuery []string, channel chan Events) error {
	// check if job id already exists (return error if it does)
	if _, exists := c.subs[jobID]; exists {
		return fmt.Errorf("[%s] Subscription already exists", jobID)
	}

	err := c.ensureWsConnection()
	if err != nil {
		return err
	}

	// send subscribe payload
	payload := request{
		Jsonrpc: "2.0",
		Method:  "subscribe",
		Params:  filterQuery,
		ID:      jobID,
	}
	// save response channel to subscription
	c.subs[jobID] = subscription{Channel: channel, Payload: payload}

	// set up chanel to catch first response
	subUnsub := make(chan Events)
	c.subUnsub[jobID] = subUnsub

	// send payload
	if err := c.ws.conn.WriteJSON(payload); err != nil {
		return err
	}

	// wait for successful subscription message
	msg := <-subUnsub
	if msg.SubErr != "" {
		return fmt.Errorf("[%s] WS error: %s", jobID, msg.SubErr)
	}
	return nil
}

// listen is the message
func (c *Client) listen() {
	// TODO: Need a way to prevent "use of closed network connection" error when closing connection
	// https://github.com/smartcontractkit/chainlink-terra/issues/25
	for {
		_, messageBytes, err := c.ws.conn.ReadMessage()
		if err != nil {
			c.Log.Errorf("[client/listen/read]: %s", err)
			return
		}
		message := string(messageBytes)
		// parse job id
		jobID := gjson.Get(message, "id")

		// if in a sub or unsub state, return error string
		if c.subUnsub[jobID.Str] != nil {
			error := gjson.Get(message, "error")
			c.subUnsub[jobID.Str] <- Events{SubErr: error.Raw}

			// remove sub/unsub state
			close(c.subUnsub[jobID.Str])
			c.subUnsub[jobID.Str] = nil
			continue
		}

		// process head tracker event
		if jobID.Str == "head-tracker" {
			height := gjson.Get(message, "result.data.value.block.header.height")
			c.subs[jobID.Str].Channel <- Events{Block: height.Uint()}
			continue
		}

		if jobID.Str == string(ABCI) {
			var res abci.ResponseQuery
			// code == 0 when no error is encountered
			result := gjson.Get(message, "result.response").Raw
			err := json.Unmarshal([]byte(result), &res)
			if err != nil {
				err := fmt.Sprintf("Couldn't decode result string: %s", err)
				c.queryCh <- RpcMessage{Error: err}
				continue
			}
			if res.Code != 0 {
				// if there's an error it's the `log` field
				c.queryCh <- RpcMessage{Error: res.Log}
				continue
			}
			c.queryCh <- RpcMessage{Data: res.Value}
			continue
		}

		if jobID.Str == string(TX) {
			errorMessage := gjson.Get(message, "error")
			if errorMessage.Raw != "" {
				c.queryCh <- RpcMessage{Error: errorMessage.Raw}
				continue
			}

			response := gjson.Get(message, "result")

			c.queryCh <- RpcMessage{Data: []byte(response.Raw)}
			continue
		}

		// parse events but skip if nothing is found (happens for sub/unsub responses)
		eventsRaw := gjson.Get(message, "result.data.value.TxResult.result.events")
		if !eventsRaw.Exists() {
			continue
		}

		// parse events as tendermint events
		var events []types.Event
		if err := json.Unmarshal([]byte(eventsRaw.Raw), &events); err != nil {
			c.Log.Errorf("[client/listen/unmarshal]: %s %s", err, eventsRaw)
		}

		// parse data into a standard format based on events
		parsedEvents := parseEvents(events)

		// send data to job specific channel
		c.subs[jobID.Str].Channel <- Events{Events: parsedEvents}
	}
}

func parseEvents(events []types.Event) (output []string) {
	// PLACEHOLDER - just returns an array of event names (no data)
	// example: https://github.com/smartcontractkit/external-initiator/blob/84cec9a579530db29ae3ca2489819c3e54d4960c/blockchain/terra/terra.go#L131
	// TODO: implement OCR specific event filtering
	for _, event := range events {
		if strings.HasPrefix(event.Type, "wasm-") {
			output = append(output, event.Type)
		}
	}
	return
}

func (c *Client) Unsubscribe(ctx context.Context, jobID string) error {
	// check if jobID exists
	if _, ok := c.subs[jobID]; !ok {
		return fmt.Errorf("[%s] Cannot unsubscribe. Job does not exist", jobID)
	}

	// create unsubscribe message from the stored subscribe message
	payload := c.subs[jobID].Payload
	payload.Method = "unsubscribe"

	// set up chanel to catch unsub response
	subUnsub := make(chan Events)
	c.subUnsub[jobID] = subUnsub

	// send unsub payload
	if err := c.ws.conn.WriteJSON(payload); err != nil {
		return err
	}

	// wait for successful unsubscription message
	msg := <-subUnsub
	if msg.SubErr != "" {
		return fmt.Errorf("[%s] WS error: %s", jobID, msg.SubErr)
	}

	// remove saved job
	delete(c.subs, jobID)
	c.Log.Infof("[%s] Unsubscribe successful", jobID)
	return nil
}
*/
