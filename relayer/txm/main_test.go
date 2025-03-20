package txm

import (
	"os"
	"testing"

	"github.com/smartcontractkit/chainlink-cosmos/relayer/params"
)

func TestMain(m *testing.M) {
	params.InitCosmosSdk(
		/* bech32Prefix= */ "wasm",
		/* token= */ "cosm",
	)
	code := m.Run()
	os.Exit(code)
}
