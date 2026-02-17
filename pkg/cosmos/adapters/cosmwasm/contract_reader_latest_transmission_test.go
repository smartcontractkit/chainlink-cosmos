package cosmwasm

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	clientmocks "github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/client/mocks"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2/types"
)

func TestLatestTransmissionDetails_ParsesArrayDigest(t *testing.T) {
	reader := clientmocks.NewReaderWriter(t)
	addr := sdk.AccAddress([]byte("ocr-contract"))
	lggr := logger.Test(t)
	ocrReader := NewOCR2Reader(addr, reader, lggr)

	rawDigest := make([]uint8, 32)
	for i := range rawDigest {
		rawDigest[i] = uint8(i + 1)
	}
	numericDigest := make([]uint16, len(rawDigest))
	for i := range rawDigest {
		numericDigest[i] = uint16(rawDigest[i])
	}
	response, err := json.Marshal(map[string]any{
		"latest_config_digest": numericDigest,
		"epoch":                3,
		"round":                2,
		"latest_answer":        "5",
		"latest_timestamp":     123,
	})
	require.NoError(t, err)

	reader.On("ContractState", mock.Anything, addr, []byte(`{"latest_transmission_details":{}}`)).
		Return(response, nil).
		Once()

	digest, epoch, round, answer, ts, err := ocrReader.LatestTransmissionDetails(context.Background())
	require.NoError(t, err)

	expected, err := ocrtypes.BytesToConfigDigest(rawDigest)
	require.NoError(t, err)
	require.Equal(t, expected, digest)
	require.Equal(t, uint32(3), epoch)
	require.Equal(t, uint8(2), round)
	require.Equal(t, big.NewInt(5), answer)
	require.Equal(t, time.Unix(123, 0), ts)
}

func TestLatestTransmissionDetails_FallbackParsesArrayDigest(t *testing.T) {
	reader := clientmocks.NewReaderWriter(t)
	addr := sdk.AccAddress([]byte("ocr-contract"))
	lggr := logger.Test(t)
	ocrReader := NewOCR2Reader(addr, reader, lggr)

	rawDigest := make([]uint8, 32)
	for i := range rawDigest {
		rawDigest[i] = uint8(i + 1)
	}
	numericDigest := make([]uint16, len(rawDigest))
	for i := range rawDigest {
		numericDigest[i] = uint16(rawDigest[i])
	}
	fallbackResponse, err := json.Marshal(map[string]any{
		"scan_logs":     false,
		"config_digest": numericDigest,
		"epoch":         0,
	})
	require.NoError(t, err)

	reader.On("ContractState", mock.Anything, addr, []byte(`{"latest_transmission_details":{}}`)).
		Return([]byte(nil), errors.New("rpc error: code = Unknown desc = ocr2::state::Transmission not found")).
		Once()
	reader.On("ContractState", mock.Anything, addr, []byte(`{"latest_config_digest_and_epoch":{}}`)).
		Return(fallbackResponse, nil).
		Once()

	digest, epoch, round, answer, ts, err := ocrReader.LatestTransmissionDetails(context.Background())
	require.NoError(t, err)

	expected, err := ocrtypes.BytesToConfigDigest(rawDigest)
	require.NoError(t, err)
	require.Equal(t, expected, digest)
	require.Equal(t, uint32(0), epoch)
	require.Equal(t, uint8(0), round)
	require.Equal(t, 0, answer.Cmp(big.NewInt(0)))
	require.Equal(t, time.Unix(0, 0), ts)
}
