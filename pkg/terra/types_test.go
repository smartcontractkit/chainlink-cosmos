package terra

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteArray(t *testing.T) {
	var input ByteArray
	input = []byte("some test value")
	output, err := json.Marshal(input)

	// check if it has become a stringified array
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(output), "["))
	assert.True(t, strings.Contains(string(output), "]"))
	assert.True(t, strings.Contains(string(output), ","))

	// check if it can be parsed back correctly
	var parse []uint8
	err = json.Unmarshal(output, &parse)
	assert.NoError(t, err)
	assert.Equal(t, string(input), string(parse))
}

func TestByteArrayArray(t *testing.T) {
	var input ByteArrayArray
	input = [][]byte{[]byte("some test value"), []byte("some other value")}
	output, err := json.Marshal(input)

	// check if it has become a stringified array of arrays
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(output), "["))
	assert.True(t, strings.Contains(string(output), "]"))
	assert.True(t, strings.Contains(string(output), "],["))

	// check if it can be parsed back correctly
	var parse [][]uint8
	err = json.Unmarshal(output, &parse)
	assert.NoError(t, err)
	for i, v := range parse {
		assert.Equal(t, string(input[i]), string(v))
	}
}
