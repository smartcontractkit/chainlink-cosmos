package monitoring

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/monitoring/fcdclient"
	fcdclientmocks "github.com/smartcontractkit/chainlink-cosmos/pkg/monitoring/fcdclient/mocks"
)

func TestTxResultsSource(t *testing.T) {
	t.Run("should correcly count failed and succeeded transactions", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		chainConfig := generateChainConfig()
		feedConfig := generateFeedConfig()

		fcdClient := new(fcdclientmocks.Client)
		factory := NewTxResultsSourceFactory(fcdClient)
		source, err := factory.NewSource(chainConfig, feedConfig)
		require.NoError(t, err)

		// Setup mocks
		getTxsRaw, err := os.ReadFile("./fixtures/txs.json")
		require.NoError(t, err)
		getTxsRes := fcdclient.Response{}
		require.NoError(t, json.Unmarshal(getTxsRaw, &getTxsRes))
		fcdClient.On("GetTxList",
			mock.Anything, // context
			fcdclient.GetTxListParams{Account: feedConfig.ContractAddress, Limit: 10},
		).Return(getTxsRes, nil).Once()

		// Execute Fetch()
		data, err := source.Fetch(ctx)
		require.NoError(t, err)

		// Assertions
		txResults, ok := data.(relayMonitoring.TxResults)
		require.True(t, ok)
		require.Equal(t, uint64(94), txResults.NumSucceeded)
		require.Equal(t, uint64(6), txResults.NumFailed)
	})
}
