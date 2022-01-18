package client

import (
	"os"
	"time"

	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/smartcontractkit/chainlink-terra/pkg/terra/mocks"
	"github.com/smartcontractkit/terra.go/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

func TestErrMatch(t *testing.T) {
	errStr := "rpc error: code = InvalidArgument desc = failed to execute message; message index: 0: Error parsing into type my_first_contract::msg::ExecuteMsg: unknown variant `blah`, expected `increment` or `reset`: execute wasm contract failed: invalid request"
	m := failedMsgIndexRe.FindStringSubmatch(errStr)
	require.Equal(t, 2, len(m))
	assert.Equal(t, m[1], "0")
}

func TestBatchSim(t *testing.T) {
	//if os.Getenv("TEST_CLIENT") == "" {
	//	t.Skip()
	//}
	accounts, testdir := SetupLocalTerraNode(t, "42")
	SetupLocalTerraNode(t, "42")
	tendermintURL := "http://127.0.0.1:26657"
	fcdURL := "https://fcd.terra.dev/" // TODO we can mock this

	lggr := new(mocks.Logger)
	lggr.Test(t)
	lggr.On("Infof", mock.Anything, mock.Anything, mock.Anything).Once()
	tc, err := NewClient(
		"42",
		tendermintURL,
		fcdURL,
		10,
		lggr)
	require.NoError(t, err)
	contract := DeployTestContract(t, accounts[0], accounts[0], tc, testdir, "../testdata/my_first_contract.wasm")
	var succeed sdk.Msg = &wasmtypes.MsgExecuteContract{Sender: accounts[0].Address.String(), Contract: contract.String(), ExecuteMsg: []byte(`{"reset":{"count":5}}`)}
	var fail sdk.Msg = &wasmtypes.MsgExecuteContract{Sender: accounts[0].Address.String(), Contract: contract.String(), ExecuteMsg: []byte(`{"blah":{"count":5}}`)}

	t.Run("single success", func(t *testing.T) {
		_, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		lggr.On("Infof", mock.Anything, mock.Anything).Once()
		res, err := tc.BatchSimulateUnsigned([]SimMsg{{ID: int64(1), Msg: succeed}}, sn)
		require.NoError(t, err)
		require.Equal(t, 1, len(res.Succeeded))
		assert.Equal(t, int64(1), res.Succeeded[0].ID)
		assert.Equal(t, 0, len(res.Failed))
	})

	t.Run("single failure", func(t *testing.T) {
		_, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		lggr.On("Infof", mock.Anything, mock.Anything).Once()
		lggr.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()
		res, err := tc.BatchSimulateUnsigned([]SimMsg{{ID: int64(1), Msg: fail}}, sn)
		require.NoError(t, err)
		assert.Equal(t, 0, len(res.Succeeded))
		require.Equal(t, 1, len(res.Failed))
		assert.Equal(t, int64(1), res.Failed[0].ID)
	})

	t.Run("multi failure", func(t *testing.T) {
		_, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		lggr.On("Infof", mock.Anything, mock.Anything).Once()
		lggr.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once() // retry
		lggr.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()
		res, err := tc.BatchSimulateUnsigned([]SimMsg{{ID: int64(1), Msg: succeed}, {ID: int64(2), Msg: fail}, {ID: int64(3), Msg: fail}}, sn)
		require.NoError(t, err)
		require.Equal(t, 1, len(res.Succeeded))
		assert.Equal(t, int64(1), res.Succeeded[0].ID)
		require.Equal(t, 2, len(res.Failed))
		assert.Equal(t, int64(2), res.Failed[0].ID)
		assert.Equal(t, int64(3), res.Failed[1].ID)
	})

	t.Run("multi succeed", func(t *testing.T) {
		_, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		lggr.On("Infof", mock.Anything, mock.Anything).Once()
		lggr.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()
		res, err := tc.BatchSimulateUnsigned([]SimMsg{{ID: int64(1), Msg: succeed}, {ID: int64(2), Msg: succeed}, {ID: int64(3), Msg: fail}}, sn)
		require.NoError(t, err)
		assert.Equal(t, 2, len(res.Succeeded))
		assert.Equal(t, 1, len(res.Failed))
	})

	t.Run("all succeed", func(t *testing.T) {
		_, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		lggr.On("Infof", mock.Anything, mock.Anything, mock.Anything).Once()
		res, err := tc.BatchSimulateUnsigned([]SimMsg{{ID: int64(1), Msg: succeed}, {ID: int64(2), Msg: succeed}, {ID: int64(3), Msg: succeed}}, sn)
		require.NoError(t, err)
		assert.Equal(t, 3, len(res.Succeeded))
		assert.Equal(t, 0, len(res.Failed))
	})

	t.Run("all fail", func(t *testing.T) {
		_, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		lggr.On("Infof", mock.Anything, mock.Anything, mock.Anything).Times(3)
		lggr.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Times(2) // retry
		lggr.On("Errorf", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()
		res, err := tc.BatchSimulateUnsigned([]SimMsg{{ID: int64(1), Msg: fail}, {ID: int64(2), Msg: fail}, {ID: int64(3), Msg: fail}}, sn)
		require.NoError(t, err)
		assert.Equal(t, 0, len(res.Succeeded))
		assert.Equal(t, 3, len(res.Failed))
	})
	lggr.AssertExpectations(t)
}

func TestTerraClient(t *testing.T) {
	// Local only for now, could maybe run on CI if we install terrad there?
	if os.Getenv("TEST_CLIENT") == "" {
		t.Skip()
	}
	accounts, testdir := SetupLocalTerraNode(t, "42")
	tendermintURL := "http://127.0.0.1:26657"
	fcdURL := "https://fcd.terra.dev/" // TODO we can mock this

	// https://lcd.terra.dev/swagger/#/
	// https://fcd.terra.dev/swagger
	lggr := new(mocks.Logger)
	lggr.Test(t)
	lggr.On("Infof", mock.Anything, mock.Anything, mock.Anything).Maybe()
	lggr.On("Errorf", mock.Anything, mock.Anything, mock.Anything).Maybe()
	tc, err := NewClient(
		"42",
		tendermintURL,
		fcdURL,
		10,
		lggr)
	require.NoError(t, err)
	contract := DeployTestContract(t, accounts[0], accounts[0], tc, testdir, "../testdata/my_first_contract.wasm")

	// Check gas price works
	fgp := sdk.NewDecCoinFromDec("uluna", sdk.MustNewDecFromStr("0.01"))
	require.NoError(t, err)
	gp := tc.GasPrice(fgp)
	t.Log("Recommended:", gp)
	// Should not use fallback
	assert.NotEqual(t, gp.String(), "0.01uluna")
	b, err := tc.Balance(accounts[1].Address, "uluna")
	require.NoError(t, err)
	assert.Equal(t, "100000000", b.Amount.String())

	tx, err := tc.Tx("1234")
	require.Error(t, err)
	t.Log("invalid tx", tx, err)
	// Fund a second account
	an, sn, err := tc.Account(accounts[0].Address)
	require.NoError(t, err)
	fund := msg.NewMsgSend(accounts[0].Address, accounts[1].Address, msg.NewCoins(msg.NewInt64Coin("uluna", 1)))
	gasLimit, err := tc.SimulateUnsigned([]msg.Msg{fund}, sn)
	require.NoError(t, err)
	txBytes, err := tc.CreateAndSign([]msg.Msg{fund}, an, sn, gasLimit.GasInfo.GasUsed, DefaultGasLimitMultiplier, tc.GasPrice(fgp), accounts[0].PrivateKey, 0)
	require.NoError(t, err)
	_, err = tc.Simulate(txBytes)
	require.NoError(t, err)
	resp, err := tc.Broadcast(txBytes, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	require.NoError(t, err)

	require.Equal(t, types.CodeTypeOK, resp.TxResponse.Code)

	// Note even the blocking command doesn't let you query for the tx right away
	time.Sleep(1 * time.Second)

	b, err = tc.Balance(accounts[1].Address, "uluna")
	require.NoError(t, err)
	assert.Equal(t, "100000001", b.Amount.String())

	// Ensure we can read back the tx with Query
	tr, err := tc.TxsEvents([]string{fmt.Sprintf("tx.height=%v", resp.TxResponse.Height)})
	require.NoError(t, err)
	assert.Equal(t, 1, len(tr.TxResponses))
	assert.Equal(t, resp.TxResponse.TxHash, tr.TxResponses[0].TxHash)
	// And also Tx
	getTx, err := tc.Tx(resp.TxResponse.TxHash)
	require.NoError(t, err)
	assert.Equal(t, getTx.TxResponse.TxHash, resp.TxResponse.TxHash)

	// Check getting the height works
	latestBlock, err := tc.LatestBlock()
	require.NoError(t, err)
	assert.True(t, latestBlock.Block.Header.Height > 1)

	// Query initial contract state
	count, err := tc.ContractStore(
		contract,
		[]byte(`{"get_count":{}}`),
	)
	require.NoError(t, err)
	assert.Equal(t, `{"count":0}`, string(count))
	// Query invalid state should give an error
	count, err = tc.ContractStore(
		contract,
		[]byte(`{"blah":{}}`),
	)
	require.Error(t, err)
	require.Nil(t, count)

	// Change the contract state
	rawMsg := wasmtypes.NewMsgExecuteContract(accounts[0].Address, contract, []byte(`{"reset":{"count":5}}`), sdk.Coins{})
	an, sn, err = tc.Account(accounts[0].Address)
	require.NoError(t, err)
	resp1, err := tc.SignAndBroadcast([]msg.Msg{rawMsg}, an, sn, tc.GasPrice(fgp), accounts[0].PrivateKey, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	require.NoError(t, err)
	time.Sleep(1 * time.Second)
	// Do it again so there are multiple executions
	rawMsg = wasmtypes.NewMsgExecuteContract(accounts[0].Address, contract, []byte(`{"reset":{"count":4}}`), sdk.Coins{})
	an, sn, err = tc.Account(accounts[0].Address)
	require.NoError(t, err)
	_, err = tc.SignAndBroadcast([]msg.Msg{rawMsg}, an, sn, tc.GasPrice(fgp), accounts[0].PrivateKey, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// Observe changed contract state
	count, err = tc.ContractStore(
		contract,
		[]byte(`{"get_count":{}}`),
	)
	require.NoError(t, err)
	assert.Equal(t, `{"count":4}`, string(count))

	// Check events querying works
	// TxEvents sorts in a descending manner, so latest txes are first
	ev, err := tc.TxsEvents([]string{fmt.Sprintf("wasm-reset.contract_address='%s'", contract.String())})
	require.NoError(t, err)
	require.Equal(t, 2, len(ev.TxResponses))
	foundCount := false
	foundContract := false
	for _, event := range ev.TxResponses[0].Logs[0].Events {
		if event.Type != "wasm-reset" {
			continue
		}
		for _, attr := range event.Attributes {
			if attr.Key == "count" {
				assert.Equal(t, "4", attr.Value)
				foundCount = true
			}
			if attr.Key == "contract_address" {
				assert.Equal(t, contract.String(), attr.Value)
				foundContract = true
			}
		}
	}
	assert.True(t, foundCount)
	assert.True(t, foundContract)

	// Ensure the height filtering works
	ev, err = tc.TxsEvents([]string{fmt.Sprintf("tx.height>=%d", resp1.TxResponse.Height+1), fmt.Sprintf("wasm-reset.contract_address='%s'", contract.String())})
	require.NoError(t, err)
	require.Equal(t, 1, len(ev.TxResponses))
	ev, err = tc.TxsEvents([]string{fmt.Sprintf("tx.height=%d", resp1.TxResponse.Height), fmt.Sprintf("wasm-reset.contract_address='%s'", contract)})
	require.NoError(t, err)
	require.Equal(t, 1, len(ev.TxResponses))
	for _, ev := range ev.TxResponses[0].Logs[0].Events {
		if ev.Type == "wasm-reset" {
			for _, attr := range ev.Attributes {
				t.Log(attr.Key, attr.Value)
			}
		}
	}

	t.Run("gasprice", func(t *testing.T) {
		rawMsg := wasmtypes.NewMsgExecuteContract(accounts[0].Address, contract, []byte(`{"reset":{"count":5}}`), sdk.Coins{})
		const expCodespace = sdkerrors.RootCodespace
		for _, tt := range []struct {
			name     string
			gasPrice msg.DecCoin
			expCode  uint32
		}{
			{
				"zero",
				msg.NewInt64DecCoin(gp.Denom, 0),
				sdkerrors.ErrInsufficientFee.ABCICode(),
			},
			{
				"below-min",
				msg.NewDecCoinFromDec(gp.Denom, msg.NewDecWithPrec(1, 4)),
				sdkerrors.ErrInsufficientFee.ABCICode(),
			},
			{
				"min",
				minGasPrice,
				0,
			},
			{
				"recommended",
				gp,
				0,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Log("Gas price:", tt.gasPrice)
				an, sn, err = tc.Account(accounts[0].Address)
				require.NoError(t, err)
				resp, err = tc.SignAndBroadcast([]msg.Msg{rawMsg}, an, sn, tt.gasPrice, accounts[0].PrivateKey, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
				if tt.expCode == 0 {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
				require.NotNil(t, resp)
				if tt.expCode == 0 {
					require.Equal(t, "", resp.TxResponse.Codespace)
				} else {
					require.Equal(t, expCodespace, resp.TxResponse.Codespace)
				}
				require.Equal(t, tt.expCode, resp.TxResponse.Code)
				if tt.expCode == 0 {
					time.Sleep(2 * time.Second)
					txResp, err := tc.Tx(resp.TxResponse.TxHash)
					require.NoError(t, err)
					t.Log("Fee:", txResp.Tx.GetFee())
					t.Log("Height:", txResp.TxResponse.Height)
					require.Equal(t, resp.TxResponse.TxHash, txResp.TxResponse.TxHash)
				}
			})
		}
	})
}
