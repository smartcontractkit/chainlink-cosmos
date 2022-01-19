package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"

	tmtypes "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/smartcontractkit/terra.go/msg"
	wasmtypes "github.com/terra-money/core/x/wasm/types"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/pkg/errors"
	"github.com/terra-money/core/app"

	"github.com/smartcontractkit/terra.go/key"
	"github.com/smartcontractkit/terra.go/tx"
)

//go:generate mockery --name ReaderWriter --output ./mocks/
type ReaderWriter interface {
	Writer
	Reader
}

// Only depends on the cosmos sdk types.
type Reader interface {
	GasPrice(fallback sdk.DecCoin) sdk.DecCoin
	Account(address sdk.AccAddress) (uint64, uint64, error)
	ContractStore(contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error)
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
	BatchSimulateUnsigned(msgs SimMsgs, sequence uint64) (*BatchSimResults, error)
	SimulateUnsigned(msgs []sdk.Msg, sequence uint64) (*txtypes.SimulateResponse, error)
	CreateAndSign(msgs []sdk.Msg, account uint64, sequence uint64, gasLimit uint64, gasLimitMultiplier float64, gasPrice sdk.DecCoin, signer key.PrivKey, timeoutHeight uint64) ([]byte, error)
}

var _ ReaderWriter = (*Client)(nil)

const (
	DefaultTimeout            = 5
	DefaultGasLimitMultiplier = 1.5
)

//go:generate mockery --name Logger --output ./mocks/
type Logger interface {
	Infof(format string, values ...interface{})
	Warnf(format string, values ...interface{})
	Errorf(format string, values ...interface{})
}

type Client struct {
	fcdURL                  string
	chainID                 string
	clientCtx               cosmosclient.Context
	cosmosServiceClient     txtypes.ServiceClient
	authClient              authtypes.QueryClient
	wasmClient              wasmtypes.QueryClient
	bankClient              banktypes.QueryClient
	tendermintServiceClient tmtypes.ServiceClient

	// Timeout for node interactions
	timeout time.Duration

	log Logger
}

func NewClient(chainID string,
	tendermintURL string,
	fcdURL string,
	requestTimeoutSeconds int,
	lggr Logger,
) (*Client, error) {
	if requestTimeoutSeconds <= 0 {
		requestTimeoutSeconds = DefaultTimeout
	}
	// Note rpchttp.New or rpchttp.NewWithTimeout use a (buggy) custom transport
	// which results in new connections being created per request.
	// Pass our own client here which uses a default transport and caches connections properly.
	tmClient, err := rpchttp.NewWithClient(tendermintURL, "/websocket", &http.Client{Timeout: time.Duration(requestTimeoutSeconds)})
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

	cosmosServiceClient := txtypes.NewServiceClient(clientCtx)
	authClient := authtypes.NewQueryClient(clientCtx)
	wasmClient := wasmtypes.NewQueryClient(clientCtx)
	tendermintServiceClient := tmtypes.NewServiceClient(clientCtx)
	bankClient := banktypes.NewQueryClient(clientCtx)

	return &Client{
		chainID:                 chainID,
		cosmosServiceClient:     cosmosServiceClient,
		authClient:              authClient,
		wasmClient:              wasmClient,
		tendermintServiceClient: tendermintServiceClient,
		bankClient:              bankClient,
		clientCtx:               clientCtx,
		timeout:                 time.Duration(requestTimeoutSeconds * int(time.Second)),
		fcdURL:                  fcdURL,
		log:                     lggr,
	}, nil
}

func (c *Client) Account(addr sdk.AccAddress) (uint64, uint64, error) {
	r, err := c.authClient.Account(context.Background(), &authtypes.QueryAccountRequest{Address: addr.String()})
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

func (c *Client) GasPrice(fallback msg.DecCoin) msg.DecCoin {
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

func (c *Client) ContractStore(contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error) {
	s, err := c.wasmClient.ContractStore(context.Background(), &wasmtypes.QueryContractStoreRequest{
		ContractAddress: contractAddress.String(),
		QueryMsg:        queryMsg,
	})
	if err != nil {
		return nil, err
	}
	//  Note s will be nil on err
	return s.QueryResult, err
}

// Returns in descending order (latest txes first)
// Each event is ANDed together and follows the query language defined
// https://docs.cosmos.network/master/core/events.html
// Note one current issue https://github.com/cosmos/cosmos-sdk/issues/10448
func (c *Client) TxsEvents(events []string) (*txtypes.GetTxsEventResponse, error) {
	e, err := c.cosmosServiceClient.GetTxsEvent(context.Background(), &txtypes.GetTxsEventRequest{
		Events:     events,
		Pagination: nil,
		OrderBy:    txtypes.OrderBy_ORDER_BY_DESC,
	})
	return e, err
}

func (c *Client) Tx(hash string) (*txtypes.GetTxResponse, error) {
	e, err := c.cosmosServiceClient.GetTx(context.Background(), &txtypes.GetTxRequest{
		Hash: hash,
	})
	return e, err
}

func (c *Client) LatestBlock() (*tmtypes.GetLatestBlockResponse, error) {
	return c.tendermintServiceClient.GetLatestBlock(context.Background(), &tmtypes.GetLatestBlockRequest{})
}

func (c *Client) BlockByHeight(height int64) (*tmtypes.GetBlockByHeightResponse, error) {
	return c.tendermintServiceClient.GetBlockByHeight(context.Background(), &tmtypes.GetBlockByHeightRequest{Height: height})
}

func (c *Client) CreateAndSign(msgs []sdk.Msg, account uint64, sequence uint64, gasLimit uint64, gasLimitMultiplier float64, gasPrice sdk.DecCoin, signer key.PrivKey, timeoutHeight uint64) ([]byte, error) {
	txbuilder := tx.NewTxBuilder(app.MakeEncodingConfig().TxConfig)
	err := txbuilder.SetMsgs(msgs...)
	if err != nil {
		return nil, err
	}
	gasLimitBuffered := uint64(math.Ceil(float64(gasLimit) * float64(gasLimitMultiplier)))
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

type SimMsg struct {
	ID  int64
	Msg sdk.Msg
}

type SimMsgs []SimMsg

func (simMsgs SimMsgs) GetMsgs() []sdk.Msg {
	msgs := make([]sdk.Msg, len(simMsgs))
	for i := range simMsgs {
		msgs[i] = simMsgs[i].Msg
	}
	return msgs
}

func (simMsgs SimMsgs) GetSimMsgsIDs() []int64 {
	ids := make([]int64, len(simMsgs))
	for i := range simMsgs {
		ids[i] = simMsgs[i].ID
	}
	return ids
}

type BatchSimResults struct {
	Failed    SimMsgs
	Succeeded SimMsgs
}

var failedMsgIndexRe, _ = regexp.Compile(`^.*failed to execute message; message index: (?P<Index>\d{1}):.*$`)

func (tc *Client) failedMsgIndex(err error) (bool, int) {
	if err == nil {
		return false, 0
	}

	m := failedMsgIndexRe.FindStringSubmatch(err.Error())
	if len(m) != 2 {
		return false, 0
	}
	index, err := strconv.ParseInt(m[1], 10, 32)
	if err != nil {
		return false, 0
	}
	return true, int(index)
}

func (tc *Client) BatchSimulateUnsigned(msgs SimMsgs, sequence uint64) (*BatchSimResults, error) {
	// Assumes at least one msg is present.
	// If we fail to simulate the batch, remove the offending tx
	// and try again. Repeat until we have a successful batch.
	// Keep track of failures so we can mark them as errored.
	// Note that the error from simulating indicates the first
	// msg in the slice which failed (it simply loops over the msgs
	// and simulates them one by one, breaking at the first failure).
	var succeeded []SimMsg
	var failed []SimMsg
	toSim := msgs
	for {
		tc.log.Infof("simulating %v", toSim)
		_, err := tc.SimulateUnsigned(toSim.GetMsgs(), sequence)
		containsFailure, failureIndex := tc.failedMsgIndex(err)
		if err != nil && !containsFailure {
			return nil, err
		}
		if containsFailure {
			failed = append(failed, toSim[failureIndex])
			succeeded = append(succeeded, toSim[:failureIndex]...)
			// remove offending msg and retry
			if failureIndex == len(toSim)-1 {
				// we're done, last one failed
				tc.log.Errorf("simulation error found in last msg, failure %v, index %v, err %v", toSim[failureIndex], failureIndex, err)
				break
			}
			// otherwise there may be more to sim
			tc.log.Errorf("simulation error found in a msg, retrying with %v, failure %v, index %v, err %v", toSim[failureIndex+1:], toSim[failureIndex], failureIndex, err)
			toSim = toSim[failureIndex+1:]
		} else {
			// we're done they all succeeded
			succeeded = append(succeeded, toSim...)
			break
		}
	}
	return &BatchSimResults{
		Failed:    failed,
		Succeeded: succeeded,
	}, nil
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
	s, err := c.cosmosServiceClient.Simulate(context.Background(), &txtypes.SimulateRequest{
		TxBytes: txBytes,
	})
	return s, err
}

func (c *Client) Simulate(txBytes []byte) (*txtypes.SimulateResponse, error) {
	s, err := c.cosmosServiceClient.Simulate(context.Background(), &txtypes.SimulateRequest{
		TxBytes: txBytes,
	})
	return s, err
}

func (c *Client) Broadcast(txBytes []byte, mode txtypes.BroadcastMode) (*txtypes.BroadcastTxResponse, error) {
	res, err := c.cosmosServiceClient.BroadcastTx(context.Background(), &txtypes.BroadcastTxRequest{
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
	txBytes, err := c.CreateAndSign(msgs, account, sequence, sim.GasInfo.GasUsed, DefaultGasLimitMultiplier, gasPrice, signer, 0)
	if err != nil {
		return nil, err
	}
	return c.Broadcast(txBytes, mode)
}

func (c *Client) Balance(addr sdk.AccAddress, denom string) (*sdk.Coin, error) {
	b, err := c.bankClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{Address: addr.String(), Denom: denom})
	if err != nil {
		return nil, err
	}
	return b.Balance, nil
}
