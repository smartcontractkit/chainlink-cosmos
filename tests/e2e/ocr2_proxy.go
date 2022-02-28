package e2e

import (
	"encoding/json"

	"github.com/smartcontractkit/chainlink-terra/tests/e2e/ocr2proxytypes"
	terraClient "github.com/smartcontractkit/terra.go/client"
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
	executeMsg := ocr2proxytypes.ProposeContractMsg{
		ContractAddress: addr,
	}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	return m.send(executeMsgBytes)
}

func (m *OCRv2Proxy) ConfirmContract(addr string) error {
	executeMsg := ocr2proxytypes.ConfirmContractMsg{
		ContractAddress: addr,
	}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	return m.send(executeMsgBytes)
}

func (m *OCRv2Proxy) TransferOwnership(to string) error {
	executeMsg := ocr2proxytypes.TransferOwnershipMsg{
		ToAddress: to,
	}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	return m.send(executeMsgBytes)
}

func (m *OCRv2Proxy) send(executeMsgBytes []byte) error {
	sender := m.client.DefaultWallet.AccAddress
	_, err := m.client.SendTX(terraClient.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgExecuteContract(
				sender,
				m.address,
				executeMsgBytes,
				msg.NewCoins(),
			),
		},
	}, true)
	return err
}
