package client

import (
	"context"
	"net/http"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/smartcontractkit/terra.go/tx"

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
	Tx(hash string)(*txtypes.GetTxResponse,error)
	LatestBlock() (*tmtypes.GetLatestBlockResponse, error)
	BlockByHeight(height int64) (*tmtypes.GetBlockByHeightResponse, error)
	Balance(addr sdk.AccAddress, denom string) (*sdk.Coin, error)
}

type Writer interface {
	// Assumes all msgs are for the same from address.
	// We may want to support multiple from addresses + signers if a use case arises.
	SignAndBroadcast(msgs []sdk.Msg, accountNum uint64, sequence uint64, gasPrice sdk.DecCoin, signer key.PrivKey, mode txtypes.BroadcastMode) (*txtypes.BroadcastTxResponse, error)
}

var _ ReaderWriter = (*Client)(nil)

const (
	DefaultTimeout = 5 * time.Second
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
	cosmosURL          string
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
	timeout time.Duration,
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
	tmClient, err := cosmosclient.NewClientFromNode(tendermintURL)
	if err != nil {
		return nil, err
	}
	ec := app.MakeEncodingConfig()
	clientCtx := cosmosclient.Context{}.
		WithClient(tmClient).
		WithChainID(chainID).
		WithCodec(ec.Marshaler).
		WithLegacyAmino(ec.Amino).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithInterfaceRegistry(ec.InterfaceRegistry).
		WithTxConfig(ec.TxConfig)

	if timeout == time.Duration(0) {
		timeout = DefaultTimeout
	}
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
		timeout:            timeout,
		fcdURL:             fcdURL,
		fallbackGasPrice:   fgp,
		gasLimitMultiplier: glm,
		log:                lggr,
	}, nil
}

func (c *Client) Account(addr sdk.AccAddress) (uint64, uint64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	r, err := c.ac.Account(ctx, &authtypes.QueryAccountRequest{addr.String()})
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
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	s, err := c.wc.ContractStore(ctx, &wasmtypes.QueryContractStoreRequest{
		ContractAddress: contractAddress,
		QueryMsg:        queryMsg,
	})
	return s.QueryResult, err
}

func (c *Client) TxsEvents(events []string) (*txtypes.GetTxsEventResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	e, err := c.sc.GetTxsEvent(ctx, &txtypes.GetTxsEventRequest{
		Events:     events,
		Pagination: nil,
		OrderBy:    txtypes.OrderBy_ORDER_BY_DESC,
	})
	return e, err
}

func (c *Client) Tx(hash string) (*txtypes.GetTxResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	e, err := c.sc.GetTx(ctx, &txtypes.GetTxRequest{
		Hash:     hash,
	})
	return e, err
}

func (c *Client) LatestBlock() (*tmtypes.GetLatestBlockResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.tmc.GetLatestBlock(ctx, &tmtypes.GetLatestBlockRequest{})
}

func (c *Client) BlockByHeight(height int64) (*tmtypes.GetBlockByHeightResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.tmc.GetBlockByHeight(ctx, &tmtypes.GetBlockByHeightRequest{Height: height})
}

func (c *Client) SignAndBroadcast(msgs []sdk.Msg, account uint64, sequence uint64, gasPrice sdk.DecCoin, signer key.PrivKey, mode txtypes.BroadcastMode) (*txtypes.BroadcastTxResponse, error) {
	simCtx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	txbuilder := tx.NewTxBuilder(app.MakeEncodingConfig().TxConfig)
	txbuilder.SetMsgs(msgs...)
	sig := signing.SignatureV2{
		PubKey: &secp256k1.PubKey{},
		Data: &signing.SingleSignatureData{
			SignMode: tx.SignModeDirect,
		},
		Sequence: sequence,
	}
	txbuilder.SetSignatures(sig)
	b, err := txbuilder.GetTxBytes()
	if err != nil {
		return nil, err
	}
	s, err := c.sc.Simulate(simCtx, &txtypes.SimulateRequest{
		Tx:      nil,
		TxBytes: b,
	})
	if err != nil {
		return nil, err
	}
	gasLimit := uint64(c.gasLimitMultiplier.MulInt64(int64(s.GasInfo.GasUsed)).Ceil().RoundInt64())
	txbuilder.SetGasLimit(gasLimit)
	gasFee := msg.NewCoin(gasPrice.Denom, gasPrice.Amount.MulInt64(int64(gasLimit)).Ceil().RoundInt())
	txbuilder.SetFeeAmount(sdk.NewCoins(gasFee))
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
	broadcastCtx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	res, err := c.sc.BroadcastTx(broadcastCtx, &txtypes.BroadcastTxRequest{
		TxBytes: signedTx,
		Mode:    mode,
	})
	if err != nil {
		return nil, err
	}
	if res.TxResponse == nil {
		return nil, errors.Errorf("got nil tx response")
	}
	if res.TxResponse.Code != 0 {
		return res, errors.Errorf("tx failed with error code: %d", res.TxResponse.Code)
	}
	return res, nil
}

func (c *Client) Balance(addr sdk.AccAddress, denom string) (*sdk.Coin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	b, err := c.bc.Balance(ctx, &banktypes.QueryBalanceRequest{Address: addr.String(), Denom: denom})
	if err != nil {
		return nil, err
	}
	return b.Balance, nil
}
