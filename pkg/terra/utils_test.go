package terra

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHexToByteArray(t *testing.T) {
	inputs := []struct {
		name string
		pass bool
		in   string
		out  string // parsed output or error msg
	}{
		{"success", true, "74657374", "test"},
		{"empty", true, "", ""},
		{"fail-invalid-char", false, "test", "encoding/hex: invalid byte: U+0074 't'"},
	}

	for _, i := range inputs {
		t.Run(i.name, func(t *testing.T) {
			var out []byte
			err := HexToByteArray(i.in, &out)
			if i.pass {
				assert.NoError(t, err)
				assert.Equal(t, i.out, string(out))
				return
			}
			assert.EqualError(t, err, i.out)
		})
	}
}

func TestHexToConfigDigest(t *testing.T) {
	inputs := []struct {
		name string
		pass bool
		in   string
		out  string // parsed output or error msg
	}{
		{"success", true, "7465737420636f6e66696720646967657374203332206368617273206c6f6e67", "test config digest 32 chars long"},
		{"fail-empty", false, "", "cannot convert bytes to ConfigDigest. bytes have wrong length 0"},
		{"fail-too-short", false, "7465737420636f6e", "cannot convert bytes to ConfigDigest. bytes have wrong length 8"},
		{"fail-invalid-char", false, "test", "encoding/hex: invalid byte: U+0074 't'"},
	}

	for _, i := range inputs {
		t.Run(i.name, func(t *testing.T) {
			var out types.ConfigDigest
			err := HexToConfigDigest(i.in, &out)
			if i.pass {
				assert.NoError(t, err)
				assert.Equal(t, i.out, string(out[:]))
				return
			}
			assert.EqualError(t, err, i.out)
		})
	}
}

func TestHexToArray(t *testing.T) {
	single := "7465737420636f6e66696720646967657374203332206368617273206c6f6e67"
	singleStr := "test config digest 32 chars long"
	multiple := []string{single, single, single, single, single, single}

	t.Run("success-single", func(t *testing.T) {
		var out [][]byte
		err := HexToArray(single, 32, &out, func(b []byte) interface{} {
			return b
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(out))
		assert.Equal(t, singleStr, string(out[0]))
	})

	t.Run("success-short", func(t *testing.T) {
		var out [][]byte
		err := HexToArray(single, 8, &out, func(b []byte) interface{} {
			return b
		})
		assert.NoError(t, err)
		assert.Equal(t, 4, len(out))
		for _, o := range out {
			assert.True(t, strings.Contains(singleStr, string(o)))
		}
	})

	t.Run("success", func(t *testing.T) {
		var out [][]byte
		err := HexToArray(strings.Join(multiple, ""), 32, &out, func(b []byte) interface{} {
			return b
		})
		assert.NoError(t, err)
		assert.Equal(t, len(multiple), len(out))
		for _, o := range out {
			assert.Equal(t, []byte(singleStr), o)
		}
	})

	t.Run("success-string", func(t *testing.T) {
		var out []string
		err := HexToArray(strings.Join(multiple, ""), 32, &out, func(b []byte) interface{} {
			return string(b)
		})
		assert.NoError(t, err)
		assert.Equal(t, len(multiple), len(out))
		for _, o := range out {
			assert.Equal(t, singleStr, o)
		}
	})

	t.Run("success-account", func(t *testing.T) {
		var out []types.Account
		err := HexToArray(strings.Join(multiple, ""), 32, &out, func(b []byte) interface{} {
			return types.Account(b)
		})
		assert.NoError(t, err)
		assert.Equal(t, len(multiple), len(out))
		for _, o := range out {
			assert.Equal(t, types.Account(singleStr), o)
		}
	})

	t.Run("fail-invalid-length", func(t *testing.T) {
		var out [][]byte
		err := HexToArray(single[0:62], 32, &out, func(b []byte) interface{} {
			return b
		})
		assert.EqualError(t, err, "invalid string length")
	})

	t.Run("fail-invalid-char", func(t *testing.T) {
		var out [][]byte
		err := HexToArray(single[0:63]+"t", 32, &out, func(b []byte) interface{} {
			return b
		})
		assert.EqualError(t, err, "encoding/hex: invalid byte: U+0074 't'")
	})
}

func TestRawMessageStringIntToInt(t *testing.T) {
	inputs := []struct {
		name    string
		input   json.RawMessage
		output  int
		success bool
	}{
		{
			name:    "success",
			input:   json.RawMessage(`"32"`),
			output:  32,
			success: true,
		},
		{
			name:    "fail-invalid",
			input:   json.RawMessage(`"3a"`),
			output:  32,
			success: false,
		},
		{
			name:    "fail-unmarshal",
			input:   json.RawMessage(`[]`),
			output:  32,
			success: false,
		},
	}

	for _, i := range inputs {
		t.Run(i.name, func(t *testing.T) {
			num, err := RawMessageStringIntToInt(i.input)
			if !i.success {
				assert.Error(t, err)
				return
			}

			assert.Equal(t, i.output, num)
			assert.NoError(t, err)
		})
	}
}

func TestContractConfigToOCRConfig(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectedErr bool
	}{
		{
			"valid input",
			"010000000000000000000000000000000000000000000000000de0b6b3a763ffff",
			"01000000000000000000000000000000000000000000000000000000000000000000000000000000000de0b6b3a763ffff",
			false,
		},
		{
			"invalid input",
			"0100000000000000000000000000000000000000000000000de0b6b3a763ffff", // too short
			"",
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input, err := hex.DecodeString(test.input)
			require.NoError(t, err)
			result, err := ContractConfigToOCRConfig(input)
			if test.expectedErr {
				require.Error(t, err)
			} else {
				require.Equal(t, 49, len(result))
				require.Equal(t, test.expected, hex.EncodeToString(result))
			}
		})
	}
}
