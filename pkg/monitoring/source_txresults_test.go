package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/monitoring/mocks"
	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
)

func TestTxResultsSource(t *testing.T) {
	t.Run("should correcly count failed and succeeded transactions", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		chainConfig := generateChainConfig()
		feedConfig := generateFeedConfig()

		// Setup mocks
		latestRoundDataRes1 := []byte(`{"round_id": 42, "answer": "0.54321"}`)
		latestRoundDataRes2 := []byte(`{"round_id": 47, "answer": "0.54321"}`)

		// Setup mocks.
		rpcClient := new(mocks.ChainReader)
		// Configuration
		rpcClient.On("ContractState",
			feedConfig.ContractAddress,
			[]byte(`{"latest_round_data":{}}`),
		).Return(latestRoundDataRes1, nil).Once()
		rpcClient.On("ContractState",
			feedConfig.ContractAddress,
			[]byte(`{"latest_round_data":{}}`),
		).Return(latestRoundDataRes2, nil).Once()

		// Execute Fetch()
		factory := NewTxResultsSourceFactory(rpcClient)
		source, err := factory.NewSource(chainConfig, feedConfig)
		require.NoError(t, err)
		data, err := source.Fetch(ctx)
		require.NoError(t, err)
		data, err = source.Fetch(ctx)
		require.NoError(t, err)

		// Assertions
		txResults, ok := data.(relayMonitoring.TxResults)
		require.True(t, ok)
		require.Equal(t, uint64(5), txResults.NumSucceeded)
	})
}
