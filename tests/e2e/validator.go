package e2e

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type OCRv2Validator struct {
	client  *TerraLCDClient
	address sdk.AccAddress
}

func (m *OCRv2Validator) Address() string {
	return m.address.String()
}
