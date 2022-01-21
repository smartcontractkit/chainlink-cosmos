package monitoring

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	pkgTerra "github.com/smartcontractkit/chainlink-terra/pkg/terra"
	pkgClient "github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
	pkgMocks "github.com/smartcontractkit/chainlink-terra/pkg/terra/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTerraSource(t *testing.T) {
	chainConfig := generateTerraChainConfig()
	feedConfig := generateTerraFeedConfig()

	accounts, testdir := pkgClient.SetupLocalTerraNode(t, chainConfig.ChainID)

	lggr := new(pkgMocks.Logger)
	lggr.Test(t)
	lggr.On("Infof", mock.Anything, mock.Anything, mock.Anything).Once()

	client, err := pkgClient.NewClient(
		chainConfig.TendermintURL,
		chainConfig.FCDURL,
		chainConfig.ReadTimeout,
		lggr,
	)
	require.NoError(t, err)
	<-time.After(10 * time.Second) // TODO (dru) is this needed?
	contract := pkgClient.DeployTestContract(t, accounts[0], accounts[0], client, testdir, "../terra/testdata/my_first_contract.wasm")
	_ = contract

	factory := NewTerraSourceFactory(lggr)
	source, err := factory.NewSource(chainConfig, feedConfig)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	envelopeRaw, err := source.Fetch(ctx)
	require.NoError(t, err)
	envelope, ok := envelopeRaw.(relayMonitoring.Envelope)
	require.True(t, ok)
	_ = envelope

	require.True(t, false)
}

// helpers

func generateTerraChainConfig() TerraConfig {
	return TerraConfig{
		TendermintURL: "http://127.0.0.1:26657",
		FCDURL:        "https://fcd.terra.dev/",
		NetworkName:   "terra-devnet",
		NetworkID:     "terra-devnet",
		ChainID:       "42",
		ReadTimeout:   30 * time.Second,
		PollInterval:  time.Duration(1+rand.Intn(5)) * time.Second,
	}
}

func generateTerraFeedConfig() TerraFeedConfig {
	coins := []string{"btc", "eth", "matic", "link", "avax", "ftt", "srm", "usdc", "sol", "ray"}
	coin := coins[rand.Intn(len(coins))]
	addresses := []string{
		"terra1tghjf8lcrf7ad9hjw9ap0ptxn0q5nkang9m3p4",
		"terra1wk3s8vpkxj8c08rswt8m2ur0ufe2lkxcwh7l3d",
		"terra1ddkw35crxpeddenmcjewh5dqraxsv7vwm48xak",
		"terra1gcu7jcnyh6k74f0cp95gp4lzg4k0w26shkw22l",
		"terra16huq7fzc95eyy89xsghzchde2tvucn9ahqja3j",
	}
	address := addresses[rand.Intn(len(addresses))]
	return TerraFeedConfig{
		Name:           fmt.Sprintf("%s / usd", coin),
		Path:           fmt.Sprintf("%s-usd", coin),
		Symbol:         "$",
		HeartbeatSec:   1,
		ContractType:   "ocr2",
		ContractStatus: "status",

		ContractAddressBech32: address,
		ContractAddress:       pkgTerra.MustAccAddress(address),
	}
}
