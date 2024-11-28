package monitoring

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Generators

func generateChainConfig(t *testing.T) CosmosConfig {
	address, err := sdk.AccAddressFromBech32(randBech32())
	require.NoError(t, err)
	return CosmosConfig{
		TendermintURL:    "https://some-tendermint-url.com",
		FCDURL:           "https://fcd.terra.dev",
		NetworkName:      "cosmwasm",
		NetworkID:        "cosmwasm",
		ChainID:          "1",
		ReadTimeout:      1 * time.Second,
		PollInterval:     2 * time.Second,
		LinkTokenAddress: address,
	}
}

func generateFeedConfig(t *testing.T) CosmosFeedConfig {
	coins := []string{"btc", "eth", "matic", "link", "avax", "ftt", "srm", "usdc", "sol", "ray"}
	coin := coins[rand.Intn(len(coins))]
	address, err := sdk.AccAddressFromBech32(randBech32())
	require.NoError(t, err)
	proxyAddress, err := sdk.AccAddressFromBech32(randBech32())
	require.NoError(t, err)
	return CosmosFeedConfig{
		Name:           fmt.Sprintf("%s / usd", coin),
		Path:           fmt.Sprintf("%s-usd", coin),
		Symbol:         "$",
		HeartbeatSec:   1,
		ContractType:   "ocr2",
		ContractStatus: "status",
		Multiply:       big.NewInt(1000),

		ContractAddressBech32: address.String(),
		ContractAddress:       address,
		ProxyAddressBech32:    proxyAddress.String(),
		ProxyAddress:          proxyAddress,
	}
}

// Sources

func randBech32() string {
	return sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes()).String()
}
