package e2e

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type OCRv2Flags struct {
	client  *TerraLCDClient
	address sdk.AccAddress
}

func (m *OCRv2Flags) Address() string {
	return m.address.String()
}
