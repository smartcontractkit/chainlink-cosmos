package params

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestInitCosmosSdk(t *testing.T) {
	// sdk initialized only once
	assert.NotPanics(t, func() { InitCosmosSdk("wasm", "atom") })
	assert.NotPanics(t, func() { InitCosmosSdk("notwasm", "cosmos") })
	// calling the internal implementation panics when called a second time
	assert.Panics(t, func() { initCosmosSdk("wasm", "cosmos") })

	// first call to Init wins
	sdkConfig := sdk.GetConfig()
	assert.Equal(t, sdkConfig.GetBech32AccountAddrPrefix(), "wasm")
	_, ok := sdk.GetDenomUnit("atom")
	assert.True(t, ok)
	_, ok = sdk.GetDenomUnit("cosmos")
	assert.False(t, ok)
}
