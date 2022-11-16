package e2e

import (
	"encoding/json"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/actypes"
	terraClient "github.com/smartcontractkit/terra.go/client"
)

type AccessController struct {
	client  *TerraLCDClient
	address sdk.AccAddress
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
		Msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{
				Sender:   sender.String(),
				Contract: t.address.String(),
				Msg:      executeMsgBytes,
				Funds:    sdk.NewCoins(),
			},
		},
	}, true)
	if err != nil {
		return err
	}
	return nil
}

func (t *AccessController) RemoveAccess(addr string) error {
	fromAddr := t.client.DefaultWallet.AccAddress
	toAddr, _ := sdk.AccAddressFromHex(addr)
	executeMsg := actypes.ExecuteRemoveAccessMsg{
		RemoveAccess: actypes.ExecuteRemoveAccessTypeMsg{
			Address: toAddr,
		}}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	_, err = t.client.SendTX(terraClient.CreateTxOptions{
		Msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{
				Sender:   fromAddr.String(),
				Contract: t.address.String(),
				Msg:      executeMsgBytes,
				Funds:    sdk.NewCoins(),
			},
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
