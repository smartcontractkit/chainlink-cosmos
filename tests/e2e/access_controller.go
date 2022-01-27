package e2e

import (
	"encoding/json"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/actypes"
	terraClient "github.com/smartcontractkit/terra.go/client"
	"github.com/smartcontractkit/terra.go/msg"
)

type AccessController struct {
	client  *TerraLCDClient
	address msg.AccAddress
}

func (t *AccessController) AddAccess(addr string) error {
	sender := t.client.DefaultWallet.AccAddress
	executeMsg := actypes.ExecuteAddAccessMsg{
		AddAccess: actypes.ExecuteAddAccessTypeMsg{
			Address: sender,
		}}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	_, err = t.client.SendTX(terraClient.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgExecuteContract(
				sender,
				t.address,
				executeMsgBytes,
				msg.NewCoins(),
			),
		},
	}, true)
	if err != nil {
		return err
	}
	return nil
}

func (t *AccessController) RemoveAccess(addr string) error {
	fromAddr, _ := msg.AccAddressFromHex(t.client.DefaultWallet.Address.String())
	toAddr, _ := msg.AccAddressFromHex(addr)
	executeMsg := actypes.ExecuteRemoveAccessMsg{
		RemoveAccess: actypes.ExecuteRemoveAccessTypeMsg{
			Address: toAddr,
		}}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	_, err = t.client.SendTX(terraClient.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgExecuteContract(
				fromAddr,
				t.address,
				executeMsgBytes,
				msg.NewCoins(),
			),
		},
	}, true)
	if err != nil {
		return err
	}
	return nil
}

func (t *AccessController) HasAccess(to string) (bool, error) {
	panic("implement me")
}

func (t *AccessController) Address() string {
	return t.address.String()
}
