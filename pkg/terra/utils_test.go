package terra

import (
	"testing"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
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
