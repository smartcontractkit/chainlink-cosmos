package client

import (
	"testing"
	"time"

	"github.com/smartcontractkit/chainlink-terra/pkg/terra/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFCDEstimator(t *testing.T) {
	lggr := new(mocks.Logger)
	lggr.Test(t)
	// Note this test runs in CI against a real api, we do want to know if this API changes or becomes slow
	gpe, err := NewFCDGasPriceEstimator("https://fcd.terra.dev:443/v1/txs/gas_prices", 10*time.Second, lggr)
	require.NoError(t, err)
	assert.Equal(t, 1, len(gpe.MustGasPrice([]string{"uluna"})))
	// TODO: more testing
}
