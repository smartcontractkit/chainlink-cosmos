package monitoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"github.com/stretchr/testify/require"
)

func TestTxResultsSource(t *testing.T) {
	rawTxs, err := os.ReadFile("./fixtures/txs.json")
	require.NoError(t, err, "should be able to read fixtures")

	t.Run("should correcly count failed and succeeded transactions", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, err := writer.Write(rawTxs)
			require.NoError(t, err)
		}))
		defer srv.Close()

		chainConfig := generateChainConfig()
		chainConfig.FCDURL = srv.URL
		feedConfig := generateFeedConfig()

		factory := NewTxResultsSourceFactory(newNullLogger())
		source, err := factory.NewSource(chainConfig, feedConfig)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		data, err := source.Fetch(ctx)
		require.NoError(t, err)
		txResults, ok := data.(relayMonitoring.TxResults)
		require.True(t, ok)
		require.Equal(t, uint64(94), txResults.NumSucceeded)
		require.Equal(t, uint64(6), txResults.NumFailed)
	})
}
