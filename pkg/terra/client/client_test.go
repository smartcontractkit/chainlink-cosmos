package client

import (
	"time"

	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/smartcontractkit/chainlink-terra/pkg/terra/mocks"
	"github.com/smartcontractkit/terra.go/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
	wasmtypes "github.com/terra-money/core/x/wasm/types"
)

func TestTerraClient(t *testing.T) {
	// Local only for now, could maybe run on CI if we install terrad there?
	//if os.Getenv("TEST_CLIENT") == "" {
	//	t.Skip()
	//}
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
		"0.01",
		"1.3",
		tendermintURL,
		fcdURL,
		10*time.Second,
		lggr)
	require.NoError(t, err)
	contract := DeployTestContract(t, accounts[0], accounts[0], tc,  testdir, "../testdata/my_first_contract.wasm")


	time.Sleep(5 * time.Second)

	// Check gas price works
	gp := tc.GasPrice()
	t.Log("Recommended:", gp)
	// Should not use fallback
	assert.NotEqual(t, gp.String(), "0.01uluna")
	b, err := tc.Balance(accounts[1].Address, "uluna")
	require.NoError(t, err)
	assert.Equal(t, "100000000", b.Amount.String())

	// Fund a second account
	an, sn, err := tc.Account(accounts[0].Address)
	require.NoError(t, err)
	resp, err := tc.SignAndBroadcast([]msg.Msg{msg.NewMsgSend(accounts[0].Address, accounts[1].Address, msg.NewCoins(msg.NewInt64Coin("uluna", 1)))},
		an, sn, tc.GasPrice(), accounts[0].PrivateKey, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
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

	// Check getting the height works
	latestBlock, err := tc.LatestBlock()
	require.NoError(t, err)
	assert.True(t, latestBlock.Block.Header.Height > 1)

	// Query initial contract state
	//contract := GetContractAddr(t, tc, deploymentHash)
	count, err := tc.ContractStore(
		contract.String(),
		[]byte(`{"get_count":{}}`),
	)
	require.NoError(t, err)
	assert.Equal(t, `{"count":0}`, string(count))

	// Change the contract state
	rawMsg := wasmtypes.NewMsgExecuteContract(accounts[0].Address, contract, []byte(`{"reset":{"count":5}}`), sdk.Coins{})
	an, sn, err = tc.Account(accounts[0].Address)
	require.NoError(t, err)
	_, err = tc.SignAndBroadcast([]msg.Msg{rawMsg}, an, sn, tc.GasPrice(), accounts[0].PrivateKey, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	// Observe changed contract state
	count, err = tc.ContractStore(
		contract.String(),
		[]byte(`{"get_count":{}}`),
	)
	require.NoError(t, err)
	assert.Equal(t, `{"count":5}`, string(count))

	t.Run("gasprice", func(t *testing.T) {
		rawMsg := wasmtypes.NewMsgExecuteContract(accounts[0].Address, contract, []byte(`{"reset":{"count":5}}`), sdk.Coins{})
		const expCodespace = errors.RootCodespace
		for _, tt := range []struct {
			name     string
			gasPrice msg.DecCoin
			expCode  uint32
		}{
			{
				"zero",
				msg.NewInt64DecCoin(gp.Denom, 0),
				errors.ErrInsufficientFee.ABCICode(),
			},
			{
				"below-min",
				msg.NewDecCoinFromDec(gp.Denom, msg.NewDecWithPrec(1, 4)),
				errors.ErrInsufficientFee.ABCICode(),
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
				resp, err = tc.SignAndBroadcast([]msg.Msg{rawMsg}, an,sn, tt.gasPrice, accounts[0].PrivateKey, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
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
