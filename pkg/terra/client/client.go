package client

import (
	"context"
	"net/http"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/smartcontractkit/terra.go/tx"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"

	tmtypes "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/smartcontractkit/terra.go/msg"
	wasmtypes "github.com/terra-money/core/x/wasm/types"

	"encoding/json"
	"fmt"
	"io/ioutil"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/pkg/errors"
	"github.com/terra-money/core/app"

	"time"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/smartcontractkit/terra.go/key"
)

//go:generate mockery --name ReaderWriter --output ./mocks/
type ReaderWriter interface {
	Writer
	Reader
}

// Only depends on the cosmos sdk types.
type Reader interface {
	GasPrice() sdk.DecCoin
	Account(address sdk.AccAddress) (uint64, uint64, error)
	ContractStore(contractAddress string, queryMsg []byte) ([]byte, error)
	TxsEvents(events []string) (*txtypes.GetTxsEventResponse, error)
	Tx(hash string) (*txtypes.GetTxResponse, error)
	LatestBlock() (*tmtypes.GetLatestBlockResponse, error)
	BlockByHeight(height int64) (*tmtypes.GetBlockByHeightResponse, error)
	Balance(addr sdk.AccAddress, denom string) (*sdk.Coin, error)
}

type Writer interface {
	// Assumes all msgs are for the same from address.
	// We may want to support multiple from addresses + signers if a use case arises.
	SignAndBroadcast(msgs []sdk.Msg, accountNum uint64, sequence uint64, gasPrice sdk.DecCoin, signer key.PrivKey, mode txtypes.BroadcastMode) (*txtypes.BroadcastTxResponse, error)
	Broadcast(txBytes []byte, mode txtypes.BroadcastMode) (*txtypes.BroadcastTxResponse, error)
	Simulate(txBytes []byte) (*txtypes.SimulateResponse, error)
	SimulateUnsigned(msgs []sdk.Msg, sequence uint64) (*txtypes.SimulateResponse, error)
	CreateAndSign(msgs []sdk.Msg, account uint64, sequence uint64, gasLimit uint64, gasPrice sdk.DecCoin, signer key.PrivKey, timeoutHeight uint64) ([]byte, error)
}

var _ ReaderWriter = (*Client)(nil)

const (
	DefaultTimeout = 5
)

type Logger interface {
	Infof(format string, values ...interface{})
	Warnf(format string, values ...interface{})
	Errorf(format string, values ...interface{})
}

type Client struct {
	codec *codec.LegacyAmino

	fallbackGasPrice   sdk.Dec
	gasLimitMultiplier sdk.Dec
	fcdURL             string
	chainID            string
	clientCtx          cosmosclient.Context
	sc                 txtypes.ServiceClient
	ac                 authtypes.QueryClient
	wc                 wasmtypes.QueryClient
	bc                 banktypes.QueryClient
	tmc                tmtypes.ServiceClient

	// Timeout for node interactions
	timeout time.Duration

	log Logger
}

func NewClient(chainID string,
	fallbackGasPrice string,
	gasLimitMultiplier string,
	tendermintURL string,
	fcdURL string,
	timeoutSeconds int,
	lggr Logger,
) (*Client, error) {
	fgp, err := sdk.NewDecFromStr(fallbackGasPrice)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid fallback gas price %v", fallbackGasPrice)
	}
	glm, err := sdk.NewDecFromStr(gasLimitMultiplier)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid gas limit multiplier %v", gasLimitMultiplier)
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = DefaultTimeout
	}
	tmClient, err := rpchttp.NewWithTimeout(tendermintURL, "/websocket", uint(timeoutSeconds))
	if err != nil {
		return nil, err
	}
	ec := app.MakeEncodingConfig()
	// Note should terra nodes start exposing grpc, its preferable
	// to connect directly with grpc.Dial to avoid using clientCtx (according to tendermint team).
	// If so then we would start putting timeouts on the ctx we pass in to the generate grpc client calls.
	clientCtx := cosmosclient.Context{}.
		WithClient(tmClient).
		WithChainID(chainID).
		WithCodec(ec.Marshaler).
		WithLegacyAmino(ec.Amino).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithInterfaceRegistry(ec.InterfaceRegistry).
		WithTxConfig(ec.TxConfig)

	sc := txtypes.NewServiceClient(clientCtx)
	ac := authtypes.NewQueryClient(clientCtx)
	wc := wasmtypes.NewQueryClient(clientCtx)
	tmc := tmtypes.NewServiceClient(clientCtx)
	bc := banktypes.NewQueryClient(clientCtx)

	return &Client{
		codec:              codec.NewLegacyAmino(),
		chainID:            chainID,
		sc:                 sc,
		ac:                 ac,
		wc:                 wc,
		tmc:                tmc,
		bc:                 bc,
		clientCtx:          clientCtx,
		timeout:            time.Duration(timeoutSeconds * int(time.Second)),
		fcdURL:             fcdURL,
		fallbackGasPrice:   fgp,
		gasLimitMultiplier: glm,
		log:                lggr,
	}, nil
}

func (c *Client) Account(addr sdk.AccAddress) (uint64, uint64, error) {
	r, err := c.ac.Account(context.Background(), &authtypes.QueryAccountRequest{Address: addr.String()})
	if err != nil {
		return 0, 0, err
	}
	var a authtypes.AccountI
	err = c.clientCtx.InterfaceRegistry.UnpackAny(r.Account, &a)
	if err != nil {
		return 0, 0, err
	}
	return a.GetAccountNumber(), a.GetSequence(), nil
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

func (c *Client) ContractStore(contractAddress string, queryMsg []byte) ([]byte, error) {
	s, err := c.wc.ContractStore(context.Background(), &wasmtypes.QueryContractStoreRequest{
		ContractAddress: contractAddress,
		QueryMsg:        queryMsg,
	})
	return s.QueryResult, err
}

// Returns in descending order (latest txes first)
// Each event is ANDed together and follows the query language defined
// https://docs.cosmos.network/master/core/events.html
// Note one current issue https://github.com/cosmos/cosmos-sdk/issues/10448
func (c *Client) TxsEvents(events []string) (*txtypes.GetTxsEventResponse, error) {
	e, err := c.sc.GetTxsEvent(context.Background(), &txtypes.GetTxsEventRequest{
		Events:     events,
		Pagination: nil,
		OrderBy:    txtypes.OrderBy_ORDER_BY_DESC,
	})
	return e, err
}

func (c *Client) Tx(hash string) (*txtypes.GetTxResponse, error) {
	e, err := c.sc.GetTx(context.Background(), &txtypes.GetTxRequest{
		Hash: hash,
	})
	return e, err
}

func (c *Client) LatestBlock() (*tmtypes.GetLatestBlockResponse, error) {
	return c.tmc.GetLatestBlock(context.Background(), &tmtypes.GetLatestBlockRequest{})
}

func (c *Client) BlockByHeight(height int64) (*tmtypes.GetBlockByHeightResponse, error) {
	return c.tmc.GetBlockByHeight(context.Background(), &tmtypes.GetBlockByHeightRequest{Height: height})
}

func (c *Client) CreateAndSign(msgs []sdk.Msg, account uint64, sequence uint64, gasLimit uint64, gasPrice sdk.DecCoin, signer key.PrivKey, timeoutHeight uint64) ([]byte, error) {
	txbuilder := tx.NewTxBuilder(app.MakeEncodingConfig().TxConfig)
	err := txbuilder.SetMsgs(msgs...)
	if err != nil {
		return nil, err
	}
	gasLimitBuffered := uint64(c.gasLimitMultiplier.MulInt64(int64(gasLimit)).Ceil().RoundInt64())
	txbuilder.SetGasLimit(gasLimitBuffered)
	gasFee := msg.NewCoin(gasPrice.Denom, gasPrice.Amount.MulInt64(int64(gasLimitBuffered)).Ceil().RoundInt())
	txbuilder.SetFeeAmount(sdk.NewCoins(gasFee))
	// 0 timeout height means unset.
	txbuilder.SetTimeoutHeight(timeoutHeight)
	err = txbuilder.Sign(tx.SignModeDirect, tx.SignerData{
		AccountNumber: account,
		ChainID:       c.chainID,
		Sequence:      sequence,
	}, signer, true)
	if err != nil {
		return nil, err
	}
	signedTx, err := txbuilder.GetTxBytes()
	if err != nil {
		return nil, err
	}
	return signedTx, nil
}

func (c *Client) SimulateUnsigned(msgs []sdk.Msg, sequence uint64) (*txtypes.SimulateResponse, error) {
	txbuilder := tx.NewTxBuilder(app.MakeEncodingConfig().TxConfig)
	if err := txbuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}
	// Create an empty signature literal as the ante handler will populate with a
	// sentinel pubkey.
	sig := signing.SignatureV2{
		PubKey: &secp256k1.PubKey{},
		Data: &signing.SingleSignatureData{
			SignMode: tx.SignModeDirect,
		},
		Sequence: sequence,
	}
	if err := txbuilder.SetSignatures(sig); err != nil {
		return nil, err
	}
	txBytes, err := txbuilder.GetTxBytes()
	if err != nil {
		return nil, err
	}
	s, err := c.sc.Simulate(context.Background(), &txtypes.SimulateRequest{
		Tx:      nil,
		TxBytes: txBytes,
	})
	return s, err
}

func (c *Client) Simulate(txBytes []byte) (*txtypes.SimulateResponse, error) {
	s, err := c.sc.Simulate(context.Background(), &txtypes.SimulateRequest{
		Tx:      nil,
		TxBytes: txBytes,
	})
	return s, err
}

func (c *Client) Broadcast(txBytes []byte, mode txtypes.BroadcastMode) (*txtypes.BroadcastTxResponse, error) {
	res, err := c.sc.BroadcastTx(context.Background(), &txtypes.BroadcastTxRequest{
		Mode:    mode,
		TxBytes: txBytes,
	})
	if err != nil {
		return nil, err
	}
	if res.TxResponse == nil {
		return nil, errors.Errorf("got nil tx response")
	}
	if res.TxResponse.Code != 0 {
		return res, errors.Errorf("tx failed with error code: %d, resp %v", res.TxResponse.Code, res.TxResponse)
	}
	return res, err
}

func (c *Client) SignAndBroadcast(msgs []sdk.Msg, account uint64, sequence uint64, gasPrice sdk.DecCoin, signer key.PrivKey, mode txtypes.BroadcastMode) (*txtypes.BroadcastTxResponse, error) {
	sim, err := c.SimulateUnsigned(msgs, sequence)
	if err != nil {
		return nil, err
	}
	txBytes, err := c.CreateAndSign(msgs, account, sequence, sim.GasInfo.GasUsed, gasPrice, signer, 0)
	if err != nil {
		return nil, err
	}
	return c.Broadcast(txBytes, mode)
}

func (c *Client) Balance(addr sdk.AccAddress, denom string) (*sdk.Coin, error) {
	b, err := c.bc.Balance(context.Background(), &banktypes.QueryBalanceRequest{Address: addr.String(), Denom: denom})
	if err != nil {
		return nil, err
	}
	return b.Balance, nil
}
