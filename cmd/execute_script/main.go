package main

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/smartcontractkit/terra.go/client"
	"github.com/smartcontractkit/terra.go/key"
	"github.com/smartcontractkit/terra.go/msg"
)

func panicErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func main() {
	privKeyBz, err := key.DerivePrivKeyBz("TODO TERRA WALLET MNEMONIC", key.CreateHDPath(0, 0))
	panicErr(err)
	privKey, err := key.PrivKeyGen(privKeyBz)
	panicErr(err)

	// Double check its the expected key
	addr := msg.AccAddress(privKey.PubKey().Address())
	if addr.String() != "TODO PAYEE ADDRESS" {
		panic(addr.String())
	}

	// Create client
	c := client.NewLCDClient(
		"TODO REST ENDPOINT 1317 PORT",
		"TODO CHAINID e.g. bombay-12 etc",
		// See prices https://fcd.terra.dev/v1/txs/gas_prices
		// Can use other prices if desired (assuming the wallet holds those funds)
		msg.NewDecCoinFromDec("ucosm", msg.NewDecFromIntWithPrec(msg.NewInt(1133), 5)), // gas price of 0.01133ucosm
		msg.NewDecFromIntWithPrec(msg.NewInt(15), 1),                                   // Gas multiplier of 1.5
		privKey,
		5*time.Second,
	)

	// Create tx
	contractAddress, err := sdk.AccAddressFromBech32("TODO CONTRACT ADDRESS")
	panicErr(err)
	// Note must be transmitter associated with payee
	execute := msg.NewMsgExecuteContract(addr, contractAddress, []byte(`{"withdraw_payment":{"transmitter":"TODO TRANSMITTER ADDRESS"}}`), sdk.Coins{})
	tx, err := c.CreateAndSignTx(context.Background(), client.CreateTxOptions{
		Msgs: []msg.Msg{
			execute,
		},
		Memo: "",
		// Options Paramters (if empty, load chain info)
		// AccountNumber: msg.NewInt(33),
		// Sequence:      msg.NewInt(1),
		// Options Paramters (if empty, simulate gas & fee)
		// FeeAmount: msg.NewCoins(),
		// GasLimit: 1000000,
		// FeeGranter: msg.AccAddress{},
		// SignMode: tx.SignModeDirect,
	})
	panicErr(err)

	// Broadcast
	res, err := c.Broadcast(context.Background(), tx, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	panicErr(err)
	fmt.Println(res)
}
