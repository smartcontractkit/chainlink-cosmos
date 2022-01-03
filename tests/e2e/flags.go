package e2e

import (
	"github.com/smartcontractkit/terra.go/msg"
)

type OCRv2Flags struct {
	client  *TerraLCDClient
	address msg.AccAddress
}

func (m *OCRv2Flags) Address() string {
	return m.address.String()
}
