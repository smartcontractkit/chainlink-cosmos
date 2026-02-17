package client

import (
	"os"
	"testing"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// these are hardcoded in test_helpers.go.
	params.InitCosmosSdk(
		/* bech32Prefix= */ "wasm",
		/* token= */ "cosm",
	)
	code := m.Run()
	os.Exit(code)
}

func TestErrMatch(t *testing.T) {
	errStr := "rpc error: code = InvalidArgument desc = failed to execute message; message index: 0: Error parsing into type my_first_contract::msg::ExecuteMsg: unknown variant `blah`, expected `increment` or `reset`: execute wasm contract failed: invalid request"
	m := failedMsgIndexRe.FindStringSubmatch(errStr)
	require.Equal(t, 2, len(m))
	assert.Equal(t, m[1], "0")

	errStr = "rpc error: code = InvalidArgument desc = failed to execute message; message index: 10: Error parsing into type my_first_contract::msg::ExecuteMsg: unknown variant `blah`, expected `increment` or `reset`: execute wasm contract failed: invalid request"
	m = failedMsgIndexRe.FindStringSubmatch(errStr)
	require.Equal(t, 2, len(m))
	assert.Equal(t, m[1], "10")

	errStr = "rpc error: code = InvalidArgument desc = failed to execute message; message index: 10000: Error parsing into type my_first_contract::msg::ExecuteMsg: unknown variant `blah`, expected `increment` or `reset`: execute wasm contract failed: invalid request"
	m = failedMsgIndexRe.FindStringSubmatch(errStr)
	require.Equal(t, 2, len(m))
	assert.Equal(t, m[1], "10000")
}
