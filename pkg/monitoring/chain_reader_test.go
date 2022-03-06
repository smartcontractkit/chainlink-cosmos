package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChainReader(t *testing.T) {
	t.Run("withContext()", func(t *testing.T) {
		t.Run("should return an error when context expires", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			data, err := withContext(ctx, func() (interface{}, error) {
				<-time.After(150 * time.Millisecond)
				return "some fake value", nil
			})
			require.Error(t, context.DeadlineExceeded, err)
			require.Equal(t, nil, data)
		})
		t.Run("should return an error when context is cancelled", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() {
				<-time.After(50 * time.Millisecond)
				cancel()
			}()
			data, err := withContext(ctx, func() (interface{}, error) {
				<-time.After(150 * time.Millisecond)
				return "some fake value", nil
			})
			require.Error(t, context.DeadlineExceeded, err)
			require.Equal(t, nil, data)
		})
		t.Run("should return whatever the call returned and not leak any goroutines", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			data, err := withContext(ctx, func() (interface{}, error) {
				<-time.After(50 * time.Millisecond)
				return "some fake value", nil
			})
			require.NoError(t, err)
			require.Equal(t, "some fake value", data)
		})
	})
}
