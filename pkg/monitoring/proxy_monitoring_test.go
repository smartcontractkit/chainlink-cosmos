package monitoring

import (
	"context"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	relayMonitoring "github.com/smartcontractkit/chainlink-common/pkg/monitoring"
	"github.com/smartcontractkit/chainlink-common/pkg/utils/tests"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/monitoring/mocks"
)

func TestProxyMonitoring(t *testing.T) {
	t.Parallel()

	t.Run("the read proxied value should be reported to prometheus", func(t *testing.T) {
		// This test checks both the source and the corresponding exporter.
		// It does so by using a mock ChainReader to return values that the real proxy would return.
		// Then it uses a mock Metrics object to record the data exported to prometheus.

		ctx := tests.Context(t)

		chainConfig := generateChainConfig(t)
		feedConfig := generateFeedConfig(t)
		feedConfig.Multiply = big.NewInt(100)
		nodes := []relayMonitoring.NodeConfig{}

		chainReader := mocks.NewChainReader(t)
		metrics := mocks.NewMetrics(t)

		lggr := logger.Test(t)
		sourceFactory := NewProxySourceFactory(chainReader, lggr)
		source, err := sourceFactory.NewSource(chainConfig, feedConfig)
		require.NoError(t, err)

		exporterFactory := NewPrometheusExporterFactory(lggr, metrics)
		exporter, err := exporterFactory.NewExporter(relayMonitoring.ExporterParams{ChainConfig: chainConfig, FeedConfig: feedConfig, Nodes: nodes})
		require.NoError(t, err)

		// Setup claims.
		chainReader.On("ContractState",
			mock.Anything, // context
			feedConfig.ProxyAddress,
			[]byte(`{"latest_round_data":{}}`),
		).Return(
			[]byte(`{"round_id":5709,"answer":"2632212500","observations_timestamp":1645456354,"transmission_timestamp":1645456380}`),
			nil,
		).Once()
		metrics.On("SetProxyAnswersRaw",
			float64(2632212500),            // answer
			feedConfig.ProxyAddressBech32,  // proxyContractAddress
			feedConfig.GetID(),             // feedID
			chainConfig.GetChainID(),       // chainID
			feedConfig.GetContractStatus(), // contractStatus
			feedConfig.GetContractType(),   // contractType
			feedConfig.GetName(),           // feedName
			feedConfig.GetPath(),           // feedPath
			chainConfig.GetNetworkID(),     // networkID
			chainConfig.GetNetworkName(),   // networkName
		)
		metrics.On("SetProxyAnswers",
			float64(26322125),              // answer / multiply
			feedConfig.ProxyAddressBech32,  // proxyContractAddress
			feedConfig.GetID(),             // feedID
			chainConfig.GetChainID(),       // chainID
			feedConfig.GetContractStatus(), // contractStatus
			feedConfig.GetContractType(),   // contractType
			feedConfig.GetName(),           // feedName
			feedConfig.GetPath(),           // feedPath
			chainConfig.GetNetworkID(),     // networkID
			chainConfig.GetNetworkName(),   // networkName
		)
		metrics.On("Cleanup",
			feedConfig.ProxyAddressBech32,  // proxyContractAddress
			feedConfig.GetID(),             // feedID
			chainConfig.GetChainID(),       // chainID
			feedConfig.GetContractStatus(), // contractStatus
			feedConfig.GetContractType(),   // contractType
			feedConfig.GetName(),           // feedName
			feedConfig.GetPath(),           // feedPath
			chainConfig.GetNetworkID(),     // networkID
			chainConfig.GetNetworkName(),   // networkName
		)

		// Run the setup
		data, err := source.Fetch(ctx)
		require.NoError(t, err)
		exporter.Export(ctx, data)
		exporter.Cleanup(ctx)
	})

	t.Run("contract without a proxy are not monitored by the proxy source", func(t *testing.T) {
		chainConfig := generateChainConfig(t)
		feedConfig := generateFeedConfig(t)
		feedConfig.ProxyAddressBech32 = ""
		feedConfig.ProxyAddress = sdk.AccAddress{}

		chainReader := mocks.NewChainReader(t)

		sourceFactory := NewProxySourceFactory(chainReader, logger.Test(t))
		source, err := sourceFactory.NewSource(chainConfig, feedConfig)
		require.NoError(t, err)

		data, err := source.Fetch(context.Background())
		require.Nil(t, data)
		require.Error(t, err, relayMonitoring.ErrNoUpdate)
	})
}
