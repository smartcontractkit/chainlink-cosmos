package monitoring

import (
	"context"
	cryptoRand "crypto/rand"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	relayMonitoring "github.com/smartcontractkit/chainlink-common/pkg/monitoring"
)

// Generators

func generateChainConfig() CosmosConfig {
	address, _ := sdk.AccAddressFromBech32("wasm106x8mk9asfnptt5rqw5kx6hs8f75fseqa8rfz2")
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

func generateFeedConfig() CosmosFeedConfig {
	coins := []string{"btc", "eth", "matic", "link", "avax", "ftt", "srm", "usdc", "sol", "ray"}
	coin := coins[rand.Intn(len(coins))]
	address, _ := sdk.AccAddressFromBech32("wasm106x8mk9asfnptt5rqw5kx6hs8f75fseqa8rfz2")
	proxyAddress, _ := sdk.AccAddressFromBech32("wasm106x8mk9asfnptt5rqw5kx6hs8f75fseqa8rfz2")
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

func generateBigInt(bitSize uint8) *big.Int {
	maxBigInt := new(big.Int)
	maxBigInt.Exp(big.NewInt(2), big.NewInt(int64(bitSize)), nil).Sub(maxBigInt, big.NewInt(1))

	//Generate cryptographically strong pseudo-random between 0 - max
	num, err := cryptoRand.Int(cryptoRand.Reader, maxBigInt)
	if err != nil {
		panic(fmt.Sprintf("failed to generate a really big number: %v", err))
	}
	return num
}

func generateProxyData() ProxyData {
	return ProxyData{
		Answer: generateBigInt(128),
	}
}

// Sources

// NewFakeProxySourceFactory makes a source that generates random proxy data.
func NewFakeProxySourceFactory(log relayMonitoring.Logger) relayMonitoring.SourceFactory {
	return &fakeProxySourceFactory{log}
}

type fakeProxySourceFactory struct {
	log relayMonitoring.Logger
}

func (f *fakeProxySourceFactory) NewSource(
	_ relayMonitoring.ChainConfig,
	_ relayMonitoring.FeedConfig,
) (relayMonitoring.Source, error) {
	return &fakeProxySource{f.log}, nil
}

func (f *fakeProxySourceFactory) GetType() string {
	return "fake-proxy"
}

type fakeProxySource struct {
	log relayMonitoring.Logger
}

func (f *fakeProxySource) Fetch(ctx context.Context) (interface{}, error) {
	return generateProxyData(), nil
}

func newNullLogger() logger.Logger {
	return logger.Nop()
}

var (
	_ = newNullLogger()
	_ = generateChainConfig()
	_ = generateFeedConfig()
)
