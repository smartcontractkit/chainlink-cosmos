package main

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

// 	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/testutil"
// 	// "github.com/smartcontractkit/terra.go/client"
// 	// "github.com/smartcontractkit/terra.go/msg"
// )

// func panicErr(err error) {
// 	if err != nil {
// 		panic(err.Error())
// 	}
// }

func main() {
	// 	privKey, addr, err := testutil.CreateKeyFromMnemonic("TODO TERRA WALLET MNEMONIC")
	// 	panicErr(err)

	// 	// Double check its the expected key
	// 	if addr.String() != "TODO PAYEE ADDRESS" {
	// 		panic(addr.String())
	// 	}

	// 	// Create client
	// 	c := client.NewLCDClient(
	// 		"TODO REST ENDPOINT 1317 PORT",
	// 		"TODO CHAINID e.g. bombay-12 etc",
	// 		// See prices https://fcd.terra.dev/v1/txs/gas_prices
	// 		// Can use other prices if desired (assuming the wallet holds those funds)
	// 		sdk.NewDecCoinFromDec("ucosm", sdk.NewDecFromIntWithPrec(sdk.NewInt(1133), 5)), // gas price of 0.01133ucosm
	// 		sdk.NewDecFromIntWithPrec(sdk.NewInt(15), 1),                                   // Gas multiplier of 1.5
	// 		privKey,
	// 		5*time.Second,
	// 	)

	// 	// Create tx
	// 	contractAddress, err := sdk.AccAddressFromBech32("TODO CONTRACT ADDRESS")
	// 	panicErr(err)
	// 	// Note must be transmitter associated with payee
	// 	execute := msg.NewMsgExecuteContract(addr, contractAddress, []byte(`{"withdraw_payment":{"transmitter":"TODO TRANSMITTER ADDRESS"}}`), sdk.Coins{})
	// 	tx, err := c.CreateAndSignTx(context.Background(), client.CreateTxOptions{
	// 		Msgs: []sdk.Msg{
	// 			execute,
	// 		},
	// 		Memo: "",
	// 		// Options Parameters (if empty, load chain info)
	// 		// AccountNumber: sdk.NewInt(33),
	// 		// Sequence:      sdk.NewInt(1),
	// 		// Options Parameters (if empty, simulate gas & fee)
	// 		// FeeAmount: sdk.NewCoins(),
	// 		// GasLimit: 1000000,
	// 		// FeeGranter: sdk.AccAddress{},
	// 		// SignMode: tx.SignModeDirect,
	// 	})
	// 	panicErr(err)

	// // Broadcast
	// res, err := c.Broadcast(context.Background(), tx, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	// panicErr(err)
	// fmt.Println(res)
}
