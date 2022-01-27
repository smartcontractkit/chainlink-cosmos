package e2e

import (
	"github.com/smartcontractkit/terra.go/msg"
)

type OCRv2Proxy struct {
	client  *TerraLCDClient
	address msg.AccAddress
}

func (m *OCRv2Proxy) Address() string {
	return m.address.String()
}

func (m *OCRv2Proxy) ProposeContract(addr string) error {
	panic("implement me")
}

func (m *OCRv2Proxy) ConfirmContract(addr string) error {
	panic("implement me")
}

func (m *OCRv2Proxy) TransferOwnership(addr string) error {
	panic("implement me")
}
