package e2e

import (
	"context"
	"encoding/json"
	"math/big"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/cw20types"
	terraClient "github.com/smartcontractkit/terra.go/client"
)

type LinkToken struct {
	client  *TerraLCDClient
	address sdk.AccAddress
}

func (t *LinkToken) Approve(to string, amount *big.Int) error {
	panic("implement me")
}

func (t *LinkToken) Transfer(to string, amount *big.Int) error {
	sender := t.client.DefaultWallet.AccAddress
	linkAddrBech32, err := sdk.AccAddressFromBech32(t.Address())
	if err != nil {
		return err
	}
	toAddr, err := sdk.AccAddressFromBech32(to)
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
		Msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{
				Sender:   sender.String(),
				Contract: linkAddrBech32.String(),
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
