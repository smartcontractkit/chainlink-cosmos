package e2e

import (
	"github.com/smartcontractkit/terra.go/msg"
)

type OCRv2Validator struct {
	client  *TerraLCDClient
	address msg.AccAddress
}

func (m *OCRv2Validator) Address() string {
	return m.address.String()
}
