package e2e

import (
	"context"
	"encoding/json"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/cw20types"
	terraClient "github.com/smartcontractkit/terra.go/client"
	"github.com/smartcontractkit/terra.go/msg"
	"math/big"
)

type LinkToken struct {
	client  *TerraLCDClient
	address msg.AccAddress
}

func (t *LinkToken) Approve(to string, amount *big.Int) error {
	panic("implement me")
}

func (t *LinkToken) Transfer(to string, amount *big.Int) error {
	sender := t.client.DefaultWallet.AccAddress
	linkAddrBech32, err := msg.AccAddressFromBech32(t.Address())
	if err != nil {
		return err
	}
	toAddr, err := msg.AccAddressFromBech32(to)
	if err != nil {
		return err
	}
	executeMsg := cw20types.ExecuteTransferMsg{
		Transfer: cw20types.ExecuteTransferTypeMsg{
			Amount:    amount.String(),
			Recipient: toAddr,
		}}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err

	}
	_, err = t.client.SendTX(terraClient.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgExecuteContract(
				sender,
				linkAddrBech32,
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

func (t *LinkToken) BalanceOf(ctx context.Context, addr string) (*big.Int, error) {
	panic("implement me")
}

func (t *LinkToken) TransferAndCall(to string, amount *big.Int, data []byte) error {
	panic("implement me")
}

func (t *LinkToken) Address() string {
	return t.address.String()
}

func (t *LinkToken) Name(_ context.Context) (string, error) {
	return "LINK Token", nil
}
