package client

import (
	"errors"
	"testing"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGasPriceEstimators(t *testing.T) {
	lggr, logs := logger.TestObserved(t, zap.WarnLevel)
	assertLogsLen := func(t *testing.T, l int) func() {
		return func() { assert.Len(t, logs.TakeAll(), l) }
	}

	t.Run("fixed", func(t *testing.T) {
		gpeFixed := NewFixedGasPriceEstimator(map[string]sdk.DecCoin{
			"ucosm": sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("10")),
		})
		p, err := gpeFixed.GasPrices()
		require.NoError(t, err)
		price, ok := p["ucosm"]
		require.True(t, ok)
		assert.Equal(t, "ucosm", price.Denom)
		assert.Equal(t, "10.000000000000000000", price.Amount.String())
	})

	t.Run("caching", func(t *testing.T) {
		responses := []sdk.DecCoin{
			sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("10")),
		}
		gpe := NewClosureGasPriceEstimator(func() (map[string]sdk.DecCoin, error) {
			if len(responses) == 0 {
				return nil, errors.New("no more prices")
			}
			var price sdk.DecCoin
			price, responses = responses[0], responses[1:]
			return map[string]sdk.DecCoin{
				"ucosm": price,
			}, nil
		})
		cachedGpe := NewCachingGasPriceEstimator(gpe, lggr)

		// Fill cache
		prices, err := cachedGpe.GasPrices()
		require.NoError(t, err)

		// Use cache, no more prices returned from estimator
		t.Cleanup(assertLogsLen(t, 1))
		cachedPrices, err := cachedGpe.GasPrices()
		require.NoError(t, err)
		assert.Equal(t, prices["ucosm"], cachedPrices["ucosm"])
	})

	t.Run("closure", func(t *testing.T) {
		gpe := NewClosureGasPriceEstimator(func() (map[string]sdk.DecCoin, error) {
			return map[string]sdk.DecCoin{
				"ucosm": sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("10")),
			}, nil
		})
		p, err := gpe.GasPrices()
		require.NoError(t, err)
		price, ok := p["ucosm"]
		require.True(t, ok)
		assert.Equal(t, "ucosm", price.Denom)
		assert.Equal(t, "10.000000000000000000", price.Amount.String())
	})

	t.Run("composed", func(t *testing.T) {
		responses := []sdk.DecCoin{}
		closureGpe := NewClosureGasPriceEstimator(func() (map[string]sdk.DecCoin, error) {
			if len(responses) == 0 {
				return nil, errors.New("no more prices")
			}
			var price sdk.DecCoin
			price, responses = responses[0], responses[1:]
			return map[string]sdk.DecCoin{
				"ucosm": price,
			}, nil
		})
		cachingGpe := NewCachingGasPriceEstimator(closureGpe, lggr)
		gpeFixed := NewFixedGasPriceEstimator(map[string]sdk.DecCoin{
			"ucosm": sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("10")),
		})
		gpe := NewMustGasPriceEstimator([]GasPricesEstimator{cachingGpe, gpeFixed}, lggr)
		t.Cleanup(assertLogsLen(t, 1))
		fixedPrices := gpe.GasPrices()
		ucosm, ok := fixedPrices["ucosm"]
		assert.True(t, ok)
		assert.Equal(t, "10.000000000000000000", ucosm.Amount.String())
		// If the url starts working, it should use that.
		responses = append(responses, sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("9")))
		gpePrices := gpe.GasPrices()
		ucosm, ok = gpePrices["ucosm"]
		assert.True(t, ok)
		assert.NotEqual(t, "10.000000000000000000", ucosm.Amount.String())
	})
}

func TestInsufficientGasError(t *testing.T) {
	accounts, testdir, tendermintURL := SetupLocalCosmosNode(t, "42")
	lggr := logger.Test(t)
	tc, err := NewClient(
		"42",
		tendermintURL,
		DefaultTimeout,
		lggr)
	require.NoError(t, err)

	t.Run("identifiable insufficient fee error", func(t *testing.T) {
		contract := DeployTestContract(t, tendermintURL, "42", accounts[0], accounts[0], tc, testdir, "../testdata/my_first_contract.wasm")
		var reset sdk.Msg = &wasmtypes.MsgExecuteContract{Sender: accounts[0].Address.String(), Contract: contract.String(), Msg: []byte(`{"reset":{"count":5}}`)}

		an, sn, err := tc.Account(accounts[0].Address)
		gasLimit, err := tc.SimulateUnsigned([]sdk.Msg{reset}, sn)
		require.NoError(t, err)
		an, sn, err = tc.Account(accounts[0].Address)
		require.NoError(t, err)
		var gasLimitMultiplier float64 = 1
		gasPrice := sdk.NewDecCoinFromDec("ucosm", sdk.MustNewDecFromStr("0.0001"))
		txBytes, err := tc.CreateAndSign([]sdk.Msg{reset}, an, sn, gasLimit.GasInfo.GasUsed, gasLimitMultiplier, gasPrice, accounts[0].PrivateKey, 0)
		require.NoError(t, err)
		_, err = tc.Simulate(txBytes)
		require.NoError(t, err)
		resp, err := tc.Broadcast(txBytes, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
		require.NotNil(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.TxResponse)

		// 13 is ErrInsufficientFee
		// https://github.com/cosmos/cosmos-sdk/blob/47f46643affd7ec7978329c42bac47275ac7e1cc/types/errors/errors.go#L56
		require.Equal(t, resp.TxResponse.Code, uint32(13))
	})
}
