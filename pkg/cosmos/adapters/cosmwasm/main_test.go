package cosmwasm

import (
	"os"
	"testing"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/params"
)

func TestMain(m *testing.M) {
	params.InitCosmosSdk(
		/* bech32Prefix= */ "wasm",
	)
	params.RegisterTokenCosmosSdk("cosm")
	code := m.Run()
	os.Exit(code)
}
