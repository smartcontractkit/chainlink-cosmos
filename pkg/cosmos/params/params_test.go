package params

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCosmosSdk(t *testing.T) {
	// sdk initialized only once
	assert.NotPanics(t, func() { InitCosmosSdk("wasm") })
	assert.NotPanics(t, func() { InitCosmosSdk("notwasm") })
	// calling the internal implementation panics when called a second time
	assert.Panics(t, func() { initCosmosSdk("wasm") })

	// first call to Init wins
	sdkConfig := sdk.GetConfig()
	assert.Equal(t, sdkConfig.GetBech32AccountAddrPrefix(), "wasm")
}

func TestRegisterToken(t *testing.T) {
	// Register single token
	err := RegisterTokenCosmosSdk("atom")
	require.NoError(t, err)

	_, ok := sdk.GetDenomUnit("atom")
	assert.True(t, ok)
	_, ok = sdk.GetDenomUnit("uatom")
	assert.True(t, ok)
	_, ok = sdk.GetDenomUnit("matom")
	assert.True(t, ok)
	_, ok = sdk.GetDenomUnit("natom")
	assert.True(t, ok)
	_, ok = sdk.GetDenomUnit("cosmos")
	assert.False(t, ok)

	// Register another token
	err = RegisterTokenCosmosSdk("cosmos")
	require.NoError(t, err)
	_, ok = sdk.GetDenomUnit("atom")
	assert.True(t, ok)
	_, ok = sdk.GetDenomUnit("cosmos")
	assert.True(t, ok)

	// Registering the same token twice panics
	err = RegisterTokenCosmosSdk("atom")
	require.Error(t, err)
}
