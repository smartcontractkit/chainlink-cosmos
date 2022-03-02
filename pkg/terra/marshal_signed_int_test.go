package terra

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalSignedInt(t *testing.T) {
	var tt = []struct {
		bytesVal string
		size     uint
		expected *big.Int
	}{
		{
			"ffffffffffffffff",
			8,
			big.NewInt(-1),
		},
		{
			"fffffffffffffffe",
			8,
			big.NewInt(-2),
		},
		{
			"0000000000000000",
			8,
			big.NewInt(0),
		},
		{
			"0000000000000001",
			8,
			big.NewInt(1),
		},
		{
			"0000000000000002",
			8,
			big.NewInt(2),
		},
		{
			"00000000000000000000000000000000",
			16,
			big.NewInt(0),
		},
		{
			"00000000000000000000000000000001",
			16,
			big.NewInt(1),
		},
		{
			"00000000000000000000000000000002",
			16,
			big.NewInt(2),
		},
		{
			"ffffffffffffffffffffffffffffffff",
			16,
			big.NewInt(-1),
		},
		{
			"fffffffffffffffffffffffffffffffe",
			16,
			big.NewInt(-2),
		},
		{
			"000000000000000000000000000000000000000000000000",
			24,
			big.NewInt(0),
		},
		{
			"000000000000000000000000000000000000000000000001",
			24,
			big.NewInt(1),
		},
		{
			"000000000000000000000000000000000000000000000002",
			24,
			big.NewInt(2),
		},
		{
			"ffffffffffffffffffffffffffffffffffffffffffffffff",
			24,
			big.NewInt(-1),
		},
		{
			"fffffffffffffffffffffffffffffffffffffffffffffffe",
			24,
			big.NewInt(-2),
		},
	}
	for _, tc := range tt {
		tc := tc
		b, err := hex.DecodeString(tc.bytesVal)
		require.NoError(t, err)
		i, err := ToInt(b, tc.size)
		require.NoError(t, err)
		assert.Equal(t, i.String(), tc.expected.String())

		// Marshalling back should give use the same
		bAfter, err := ToBytes(i, tc.size)
		require.NoError(t, err)
		assert.Equal(t, tc.bytesVal, hex.EncodeToString(bAfter))
	}
}
