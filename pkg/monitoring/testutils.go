package monitoring

import (
	"context"
	cryptoRand "crypto/rand"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"github.com/smartcontractkit/terra.go/msg"
)

// Generators

func generateChainConfig() TerraConfig {
	address, _ := msg.AccAddressFromBech32("terra106x8mk9asfnptt5rqw5kx6hs8f75fseqa8rfz2")
	return TerraConfig{
		TendermintURL:    "https://some-tendermint-url.com",
		FCDURL:           "https://fcd.terra.dev",
		NetworkName:      "terra",
		NetworkID:        "terra",
		ChainID:          "1",
		ReadTimeout:      1 * time.Second,
		PollInterval:     2 * time.Second,
		LinkTokenAddress: address,
	}
}

func generateFeedConfig() TerraFeedConfig {
	coins := []string{"btc", "eth", "matic", "link", "avax", "ftt", "srm", "usdc", "sol", "ray"}
	coin := coins[rand.Intn(len(coins))]
	address, _ := msg.AccAddressFromBech32("terra106x8mk9asfnptt5rqw5kx6hs8f75fseqa8rfz2")
	proxyAddress, _ := msg.AccAddressFromBech32("terra106x8mk9asfnptt5rqw5kx6hs8f75fseqa8rfz2")
	return TerraFeedConfig{
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

type fakeProxySource struct {
	log relayMonitoring.Logger
}

func (f *fakeProxySource) Fetch(ctx context.Context) (interface{}, error) {
	return generateProxyData(), nil
}

// Logger

type nullLogger struct{}

func newNullLogger() relayMonitoring.Logger {
	return &nullLogger{}
}

func (n *nullLogger) With(args ...interface{}) relayMonitoring.Logger {
	return n
}

func (n *nullLogger) Tracew(format string, values ...interface{})    {}
func (n *nullLogger) Debugw(format string, values ...interface{})    {}
func (n *nullLogger) Infow(format string, values ...interface{})     {}
func (n *nullLogger) Warnw(format string, values ...interface{})     {}
func (n *nullLogger) Errorw(format string, values ...interface{})    {}
func (n *nullLogger) Criticalw(format string, values ...interface{}) {}
func (n *nullLogger) Panicw(format string, values ...interface{})    {}
func (n *nullLogger) Fatalw(format string, values ...interface{})    {}

var (
	_ = newNullLogger()
	_ = generateChainConfig()
	_ = generateFeedConfig()
)
