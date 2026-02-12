package cosmwasm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

func TestJSONConfigDigestUnmarshal_Array(t *testing.T) {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i + 1)
	}

	payload := struct {
		ConfigDigest JSONConfigDigest `json:"config_digest"`
	}{}

	numeric := make([]uint16, len(raw))
	for i := range raw {
		numeric[i] = uint16(raw[i])
	}
	bytesJSON, err := json.Marshal(map[string][]uint16{"config_digest": numeric})
	require.NoError(t, err)

	err = json.Unmarshal(bytesJSON, &payload)
	require.NoError(t, err)

	expected, err := types.BytesToConfigDigest(raw)
	require.NoError(t, err)
	require.Equal(t, expected, payload.ConfigDigest.ConfigDigest())
}

func TestJSONConfigDigestUnmarshal_ArrayWrongLength(t *testing.T) {
	payload := struct {
		ConfigDigest JSONConfigDigest `json:"config_digest"`
	}{}

	err := json.Unmarshal([]byte(`{"config_digest":[1,2,3]}`), &payload)
	require.ErrorContains(t, err, "cannot convert bytes to ConfigDigest. bytes have wrong length 3")
}

func TestJSONConfigDigestUnmarshal_StringCompatibility(t *testing.T) {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i + 1)
	}
	expected, err := types.BytesToConfigDigest(raw)
	require.NoError(t, err)

	encoded, err := json.Marshal(expected)
	require.NoError(t, err)

	var got JSONConfigDigest
	err = json.Unmarshal(encoded, &got)
	require.NoError(t, err)
	require.Equal(t, expected, got.ConfigDigest())
}
