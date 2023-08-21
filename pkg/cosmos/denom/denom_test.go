package denom

import (
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/params"
)

func TestMain(m *testing.M) {
	params.InitCosmosSdk(
		/* bech32Prefix= */ "wasm",
		/* token= */ "atom",
	)
	code := m.Run()
	os.Exit(code)
}

func TestConvertDecCoinToDenomRegistered(t *testing.T) {
	tests := []struct {
		coin  sdk.DecCoin
		denom string
		exp   string
	}{
		// simple conversions
		{sdk.NewDecCoin("uatom", sdk.NewInt(0)), "atom", "0atom"},
		{sdk.NewDecCoin("atom", sdk.NewInt(1)), "atom", "1atom"},
		{sdk.NewDecCoin("matom", sdk.NewInt(1)), "uatom", "1000uatom"},
		{sdk.NewDecCoin("atom", sdk.NewInt(1)), "matom", "1000matom"},
		{sdk.NewDecCoin("atom", sdk.NewInt(1)), "uatom", "1000000uatom"},
		// truncations (rounded down, remainder discarded)
		{sdk.NewDecCoin("uatom", sdk.NewInt(1)), "atom", "0atom"},
		{sdk.NewDecCoin("matom", sdk.NewInt(1)), "atom", "0atom"},
		{sdk.NewDecCoin("uatom", sdk.NewInt(1000000)), "atom", "1atom"},
		{sdk.NewDecCoin("matom", sdk.NewInt(1000000)), "atom", "1000atom"},
		{sdk.NewDecCoin("uatom", sdk.NewInt(123456789)), "atom", "123atom"},
		{sdk.NewDecCoin("matom", sdk.NewInt(123456789)), "atom", "123456atom"},
	}
	for _, tt := range tests {
		t.Run(tt.coin.String(), func(t *testing.T) {
			got, err := ConvertDecCoinToDenom(tt.coin, tt.denom)
			require.NoError(t, err)
			require.Equal(t, tt.exp, got.String())
		})
	}
}

func TestConvertDecCoinToDenomUnregistered(t *testing.T) {
	tests := []struct {
		coin      sdk.DecCoin
		denom     string
		expErrStr string
	}{
		{sdk.NewDecCoin("zatom", sdk.NewInt(1)), "atom", "source denom not registered: zatom"},
		{sdk.NewDecCoin("atom", sdk.NewInt(1)), "xatom", "destination denom not registered: xatom"},
	}
	for _, tt := range tests {
		t.Run(tt.coin.String(), func(t *testing.T) {
			_, err := ConvertDecCoinToDenom(tt.coin, tt.denom)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expErrStr)
		})
	}
}
