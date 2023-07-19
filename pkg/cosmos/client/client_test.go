package client

import (
	"fmt"
	"os"
	"testing"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/params"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
)

func TestMain(m *testing.M) {
	// these are hardcoded in test_helpers.go.
	params.InitCosmosSdk(
		/* bech32Prefix= */ "wasm",
		/* token= */ "cosm",
	)
	code := m.Run()
	os.Exit(code)
}

func TestErrMatch(t *testing.T) {
	errStr := "rpc error: code = InvalidArgument desc = failed to execute message; message index: 0: Error parsing into type my_first_contract::msg::ExecuteMsg: unknown variant `blah`, expected `increment` or `reset`: execute wasm contract failed: invalid request"
	m := failedMsgIndexRe.FindStringSubmatch(errStr)
	require.Equal(t, 2, len(m))
	assert.Equal(t, m[1], "0")

	errStr = "rpc error: code = InvalidArgument desc = failed to execute message; message index: 10: Error parsing into type my_first_contract::msg::ExecuteMsg: unknown variant `blah`, expected `increment` or `reset`: execute wasm contract failed: invalid request"
	m = failedMsgIndexRe.FindStringSubmatch(errStr)
	require.Equal(t, 2, len(m))
	assert.Equal(t, m[1], "10")

	errStr = "rpc error: code = InvalidArgument desc = failed to execute message; message index: 10000: Error parsing into type my_first_contract::msg::ExecuteMsg: unknown variant `blah`, expected `increment` or `reset`: execute wasm contract failed: invalid request"
	m = failedMsgIndexRe.FindStringSubmatch(errStr)
	require.Equal(t, 2, len(m))
	assert.Equal(t, m[1], "10000")
}

func TestBatchSim(t *testing.T) {
	accounts, testdir, tendermintURL := SetupLocalCosmosNode(t, "42")

	lggr, logs := logger.TestObserved(t, zap.WarnLevel)
	tc, err := NewClient(
		"42",
		tendermintURL,
		DefaultTimeout,
		lggr)
	require.NoError(t, err)
	assertLogsLen := func(t *testing.T, l int) func() {
		return func() { assert.Len(t, logs.TakeAll(), l) }
	}

	contract := DeployTestContract(t, tendermintURL, "42", accounts[0], accounts[0], tc, testdir, "../testdata/my_first_contract.wasm")
	var succeed sdk.Msg = &wasmtypes.MsgExecuteContract{Sender: accounts[0].Address.String(), Contract: contract.String(), Msg: []byte(`{"reset":{"count":5}}`)}
	var fail sdk.Msg = &wasmtypes.MsgExecuteContract{Sender: accounts[0].Address.String(), Contract: contract.String(), Msg: []byte(`{"blah":{"count":5}}`)}

	t.Run("single success", func(t *testing.T) {
		_, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		t.Cleanup(assertLogsLen(t, 0))
		res, err := tc.BatchSimulateUnsigned([]SimMsg{{ID: int64(1), Msg: succeed}}, sn)
		require.NoError(t, err)
		require.Equal(t, 1, len(res.Succeeded))
		assert.Equal(t, int64(1), res.Succeeded[0].ID)
		assert.Equal(t, 0, len(res.Failed))
	})

	t.Run("single failure", func(t *testing.T) {
		_, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		t.Cleanup(assertLogsLen(t, 1))
		res, err := tc.BatchSimulateUnsigned([]SimMsg{{ID: int64(1), Msg: fail}}, sn)
		require.NoError(t, err)
		assert.Equal(t, 0, len(res.Succeeded))
		require.Equal(t, 1, len(res.Failed))
		assert.Equal(t, int64(1), res.Failed[0].ID)
	})

	t.Run("multi failure", func(t *testing.T) {
		_, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		t.Cleanup(assertLogsLen(t, 2))
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
		t.Cleanup(assertLogsLen(t, 1))
		res, err := tc.BatchSimulateUnsigned([]SimMsg{{ID: int64(1), Msg: succeed}, {ID: int64(2), Msg: succeed}, {ID: int64(3), Msg: fail}}, sn)
		require.NoError(t, err)
		assert.Equal(t, 2, len(res.Succeeded))
		assert.Equal(t, 1, len(res.Failed))
	})

	t.Run("all succeed", func(t *testing.T) {
		_, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		t.Cleanup(assertLogsLen(t, 0))
		res, err := tc.BatchSimulateUnsigned([]SimMsg{{ID: int64(1), Msg: succeed}, {ID: int64(2), Msg: succeed}, {ID: int64(3), Msg: succeed}}, sn)
		require.NoError(t, err)
		assert.Equal(t, 3, len(res.Succeeded))
		assert.Equal(t, 0, len(res.Failed))
	})

	t.Run("all fail", func(t *testing.T) {
		_, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		t.Cleanup(assertLogsLen(t, 3))
		res, err := tc.BatchSimulateUnsigned([]SimMsg{{ID: int64(1), Msg: fail}, {ID: int64(2), Msg: fail}, {ID: int64(3), Msg: fail}}, sn)
		require.NoError(t, err)
		assert.Equal(t, 0, len(res.Succeeded))
		assert.Equal(t, 3, len(res.Failed))
	})
}

func TestCosmosClient(t *testing.T) {
	// Local only for now, could maybe run on CI if we install terrad there?
	accounts, testdir, tendermintURL := SetupLocalCosmosNode(t, "42")
	lggr := logger.Test(t)
	tc, err := NewClient(
		"42",
		tendermintURL,
		DefaultTimeout,
		lggr)
	require.NoError(t, err)
	gpe := NewFixedGasPriceEstimator(map[string]sdk.DecCoin{
		"ucosm": sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.01")),
	})
	contract := DeployTestContract(t, tendermintURL, "42", accounts[0], accounts[0], tc, testdir, "../testdata/my_first_contract.wasm")

	t.Run("send tx between accounts", func(t *testing.T) {
		// Assert balance before
		b, err := tc.Balance(accounts[1].Address, "ucosm")
		require.NoError(t, err)
		assert.Equal(t, "100000000", b.Amount.String())

		// Send a ucosm from one account to another and ensure balances update
		an, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		fund := banktypes.NewMsgSend(accounts[0].Address, accounts[1].Address, sdk.NewCoins(sdk.NewInt64Coin("ucosm", 1)))
		gasLimit, err := tc.SimulateUnsigned([]sdk.Msg{fund}, sn)
		require.NoError(t, err)
		gasPrices, err := gpe.GasPrices()
		require.NoError(t, err)
		txBytes, err := tc.CreateAndSign([]sdk.Msg{fund}, an, sn, gasLimit.GasInfo.GasUsed, DefaultGasLimitMultiplier, gasPrices["ucosm"], accounts[0].PrivateKey, 0)
		require.NoError(t, err)
		_, err = tc.Simulate(txBytes)
		require.NoError(t, err)
		resp, err := tc.Broadcast(txBytes, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
		require.NoError(t, err)
		require.Equal(t, types.CodeTypeOK, resp.TxResponse.Code)

		// Note even the blocking command doesn't let you query for the tx right away
		time.Sleep(1 * time.Second)

		// Assert balance changed
		b, err = tc.Balance(accounts[1].Address, "ucosm")
		require.NoError(t, err)
		assert.Equal(t, "100000001", b.Amount.String())

		// Invalid tx should error
		_, err = tc.Tx("1234")
		require.Error(t, err)

		// Ensure we can read back the tx with Query
		tr, err := tc.TxsEvents([]string{fmt.Sprintf("tx.height=%v", resp.TxResponse.Height)}, nil)
		require.NoError(t, err)
		assert.Equal(t, 1, len(tr.TxResponses))
		assert.Equal(t, resp.TxResponse.TxHash, tr.TxResponses[0].TxHash)
		// And also Tx
		getTx, err := tc.Tx(resp.TxResponse.TxHash)
		require.NoError(t, err)
		assert.Equal(t, getTx.TxResponse.TxHash, resp.TxResponse.TxHash)
	})

	t.Run("can get height", func(t *testing.T) {
		// Check getting the height works
		latestBlock, err := tc.LatestBlock()
		require.NoError(t, err)
		assert.True(t, latestBlock.Block.Header.Height > 1)
	})

	t.Run("contract event querying", func(t *testing.T) {
		// Query initial contract state
		count, err := tc.ContractState(
			contract,
			[]byte(`{"get_count":{}}`),
		)
		require.NoError(t, err)
		assert.Equal(t, `{"count":0}`, string(count))
		// Query invalid state should give an error
		count, err = tc.ContractState(
			contract,
			[]byte(`{"blah":{}}`),
		)
		require.Error(t, err)
		require.Nil(t, count)

		// Change the contract state
		rawMsg := &wasmtypes.MsgExecuteContract{
			Sender:   accounts[0].Address.String(),
			Contract: contract.String(),
			Msg:      []byte(`{"reset":{"count":5}}`),
			Funds:    sdk.Coins{},
		}
		an, sn, err := tc.Account(accounts[0].Address)
		require.NoError(t, err)
		gasPrices, err := gpe.GasPrices()
		require.NoError(t, err)
		resp1, err := tc.SignAndBroadcast([]sdk.Msg{rawMsg}, an, sn, gasPrices["ucosm"], accounts[0].PrivateKey, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
		require.NoError(t, err)
		time.Sleep(1 * time.Second)
		// Do it again so there are multiple executions
		rawMsg = &wasmtypes.MsgExecuteContract{
			Sender:   accounts[0].Address.String(),
			Contract: contract.String(),
			Msg:      []byte(`{"reset":{"count":4}}`),
			Funds:    sdk.Coins{},
		}
		an, sn, err = tc.Account(accounts[0].Address)
		require.NoError(t, err)
		_, err = tc.SignAndBroadcast([]sdk.Msg{rawMsg}, an, sn, gasPrices["ucosm"], accounts[0].PrivateKey, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
		require.NoError(t, err)
		time.Sleep(1 * time.Second)

		// Observe changed contract state
		count, err = tc.ContractState(
			contract,
			[]byte(`{"get_count":{}}`),
		)
		require.NoError(t, err)
		assert.Equal(t, `{"count":4}`, string(count))

		// Check events querying works
		// TxEvents sorts in a descending manner, so latest txes are first
		ev, err := tc.TxsEvents([]string{"wasm.action='reset'", fmt.Sprintf("wasm._contract_address='%s'", contract.String())}, nil)
		require.NoError(t, err)
		require.Equal(t, 2, len(ev.TxResponses))
		foundContract := false
		for _, event := range ev.TxResponses[0].Logs[0].Events {
			if event.Type != "wasm" {
				continue
			}
			isResetAction := false
			for _, attr := range event.Attributes {
				if attr.Key != "action" {
					continue
				}
				isResetAction = attr.Value == "reset"
				break
			}
			if !isResetAction {
				continue
			}
			for _, attr := range event.Attributes {
				if attr.Key == "_contract_address" {
					assert.Equal(t, contract.String(), attr.Value)
					foundContract = true
				}
			}
		}
		assert.True(t, foundContract)

		// Ensure the height filtering works
		ev, err = tc.TxsEvents([]string{fmt.Sprintf("tx.height>=%d", resp1.TxResponse.Height+1), "wasm.action='reset'", fmt.Sprintf("wasm._contract_address='%s'", contract.String())}, nil)
		require.NoError(t, err)
		require.Equal(t, 1, len(ev.TxResponses))
		ev, err = tc.TxsEvents([]string{fmt.Sprintf("tx.height=%d", resp1.TxResponse.Height), "wasm.action='reset'", fmt.Sprintf("wasm._contract_address='%s'", contract)}, nil)
		require.NoError(t, err)
		require.Equal(t, 1, len(ev.TxResponses))
		for _, ev := range ev.TxResponses[0].Logs[0].Events {
			if ev.Type == "wasm-reset" {
				for _, attr := range ev.Attributes {
					t.Log(attr.Key, attr.Value)
				}
			}
		}
	})

	t.Run("gasprice", func(t *testing.T) {
		rawMsg := &wasmtypes.MsgExecuteContract{
			Sender:   accounts[0].Address.String(),
			Contract: contract.String(),
			Msg:      []byte(`{"reset":{"count":5}}`),
			Funds:    sdk.Coins{},
		}
		const expCodespace = sdkerrors.RootCodespace
		gasPrices, err := gpe.GasPrices()
		require.NoError(t, err)
		for _, tt := range []struct {
			name     string
			gasPrice sdk.DecCoin
			expCode  uint32
		}{
			{
				"zero",
				sdk.NewInt64DecCoin("ucosm", 0),
				sdkerrors.ErrInsufficientFee.ABCICode(),
			},
			{
				"below-min",
				sdk.NewDecCoinFromDec("ucosm", sdk.NewDecWithPrec(1, 4)),
				sdkerrors.ErrInsufficientFee.ABCICode(),
			},
			{
				"min",
				minGasPrice,
				0,
			},
			{
				"recommended",
				gasPrices["ucosm"],
				0,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Log("Gas price:", tt.gasPrice)
				an, sn, err := tc.Account(accounts[0].Address)
				require.NoError(t, err)
				resp, err := tc.SignAndBroadcast([]sdk.Msg{rawMsg}, an, sn, tt.gasPrice, accounts[0].PrivateKey, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
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
