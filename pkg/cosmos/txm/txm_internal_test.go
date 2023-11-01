package txm

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	tmservicetypes "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"github.com/smartcontractkit/chainlink-relay/pkg/utils"
	"github.com/smartcontractkit/chainlink-relay/pkg/utils/tests"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/client"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/client/mocks"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/config"
	cosmosdb "github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/db"
)

func generateExecuteMsg(msg []byte, from, to cosmostypes.AccAddress) cosmostypes.Msg {
	return &wasmtypes.MsgExecuteContract{
		Sender:   from.String(),
		Contract: to.String(),
		Msg:      msg,
		Funds:    cosmostypes.Coins{},
	}
}

func newReaderWriterMock(t *testing.T) *mocks.ReaderWriter {
	tc := new(mocks.ReaderWriter)
	tc.Test(t)
	t.Cleanup(func() { tc.AssertExpectations(t) })
	return tc
}

func TestTxm(t *testing.T) {
	lggr := logger.Test(t)
	db := NewDB(t)
	ks := newKeystore(4)

	adapter := newKeystoreAdapter(ks, "wasm")
	accounts, err := adapter.Accounts()
	require.NoError(t, err)
	require.Equal(t, len(accounts), 4)

	sender1, err := cosmostypes.AccAddressFromBech32(accounts[0])
	require.NoError(t, err)
	sender2, err := cosmostypes.AccAddressFromBech32(accounts[1])
	require.NoError(t, err)
	contract, err := cosmostypes.AccAddressFromBech32(accounts[2])
	require.NoError(t, err)
	contract2, err := cosmostypes.AccAddressFromBech32(accounts[3])
	require.NoError(t, err)

	chainID := RandomChainID()
	two := int64(2)
	gasToken := "ucosm"
	cfg := &config.TOMLConfig{Chain: config.Chain{
		MaxMsgsPerBatch: &two,
		GasToken:        &gasToken,
	}}
	cfg.SetDefaults()
	gpe := client.NewMustGasPriceEstimator([]client.GasPricesEstimator{
		client.NewFixedGasPriceEstimator(map[string]cosmostypes.DecCoin{
			cfg.GasToken(): cosmostypes.NewDecCoinFromDec(cfg.GasToken(), cosmostypes.MustNewDecFromStr("0.01")),
		},
			lggr.(logger.SugaredLogger),
		),
	}, lggr)

	t.Run("single msg", func(t *testing.T) {
		ctx := tests.Context(t)
		tc := newReaderWriterMock(t)
		tcFn := func() (client.ReaderWriter, error) { return tc, nil }
		loopKs := newKeystore(1)
		txm := NewTxm(db, tcFn, *gpe, chainID, cfg, loopKs, lggr)

		// Enqueue a single msg, then send it in a batch
		id1, err := txm.Enqueue(ctx, contract.String(), generateExecuteMsg([]byte(`1`), sender1, contract))
		require.NoError(t, err)
		tc.On("Account", mock.Anything).Return(uint64(0), uint64(0), nil)
		tc.On("BatchSimulateUnsigned", mock.Anything, mock.Anything).Return(&client.BatchSimResults{
			Failed: nil,
			Succeeded: client.SimMsgs{{ID: id1, Msg: &wasmtypes.MsgExecuteContract{
				Sender: sender1.String(),
				Msg:    []byte(`1`),
			}}},
		}, nil)
		tc.On("SimulateUnsigned", mock.Anything, mock.Anything).Return(&txtypes.SimulateResponse{GasInfo: &cosmostypes.GasInfo{
			GasUsed: 1_000_000,
		}}, nil)
		tc.On("LatestBlock").Return(&tmservicetypes.GetLatestBlockResponse{SdkBlock: &tmservicetypes.Block{
			Header: tmservicetypes.Header{Height: 1},
		}}, nil)
		tc.On("CreateAndSign", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]byte{0x01}, nil)

		txResp := &cosmostypes.TxResponse{TxHash: "4BF5122F344554C53BDE2EBB8CD2B7E3D1600AD631C385A5D7CCE23C7785459A"}
		tc.On("Broadcast", mock.Anything, mock.Anything).Return(&txtypes.BroadcastTxResponse{TxResponse: txResp}, nil)
		tc.On("Tx", mock.Anything).Return(&txtypes.GetTxResponse{Tx: &txtypes.Tx{}, TxResponse: txResp}, nil)
		txm.sendMsgBatch(tests.Context(t))

		// Should be in completed state
		completed, err := txm.orm.GetMsgs(ctx, id1)
		require.NoError(t, err)
		require.Equal(t, 1, len(completed))
		assert.Equal(t, completed[0].State, cosmosdb.Confirmed)
	})

	t.Run("two msgs different accounts", func(t *testing.T) {
		ctx := tests.Context(t)
		tc := newReaderWriterMock(t)
		tcFn := func() (client.ReaderWriter, error) { return tc, nil }
		loopKs := newKeystore(1)
		txm := NewTxm(db, tcFn, *gpe, chainID, cfg, loopKs, lggr)

		id1, err := txm.Enqueue(ctx, contract.String(), generateExecuteMsg([]byte(`0`), sender1, contract))
		require.NoError(t, err)
		id2, err := txm.Enqueue(ctx, contract.String(), generateExecuteMsg([]byte(`1`), sender2, contract))
		require.NoError(t, err)

		tc.On("Account", mock.Anything).Return(uint64(0), uint64(0), nil).Once()
		// Note this must be arg dependent, we don't know which order
		// the procesing will happen in (map iteration by from address).
		tc.On("BatchSimulateUnsigned", client.SimMsgs{
			{
				ID: id2,
				Msg: &wasmtypes.MsgExecuteContract{
					Sender:   sender2.String(),
					Msg:      []byte(`1`),
					Contract: contract.String(),
				},
			},
		}, mock.Anything).Return(&client.BatchSimResults{
			Failed: nil,
			Succeeded: client.SimMsgs{
				{
					ID: id2,
					Msg: &wasmtypes.MsgExecuteContract{
						Sender:   sender2.String(),
						Msg:      []byte(`1`),
						Contract: contract.String(),
					},
				},
			},
		}, nil).Once()
		tc.On("SimulateUnsigned", mock.Anything, mock.Anything).Return(&txtypes.SimulateResponse{GasInfo: &cosmostypes.GasInfo{
			GasUsed: 1_000_000,
		}}, nil).Once()
		tc.On("LatestBlock").Return(&tmservicetypes.GetLatestBlockResponse{SdkBlock: &tmservicetypes.Block{
			Header: tmservicetypes.Header{Height: 1},
		}}, nil).Once()
		tc.On("CreateAndSign", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]byte{0x01}, nil).Once()
		txResp := &cosmostypes.TxResponse{TxHash: "4BF5122F344554C53BDE2EBB8CD2B7E3D1600AD631C385A5D7CCE23C7785459A"}
		tc.On("Broadcast", mock.Anything, mock.Anything).Return(&txtypes.BroadcastTxResponse{TxResponse: txResp}, nil).Once()
		tc.On("Tx", mock.Anything).Return(&txtypes.GetTxResponse{Tx: &txtypes.Tx{}, TxResponse: txResp}, nil).Once()
		txm.sendMsgBatch(tests.Context(t))

		// Should be in completed state
		completed, err := txm.orm.GetMsgs(ctx, id1, id2)
		require.NoError(t, err)
		require.Equal(t, 2, len(completed))
		assert.Equal(t, cosmosdb.Errored, completed[0].State) // cancelled
		assert.Equal(t, cosmosdb.Confirmed, completed[1].State)
	})

	t.Run("two msgs different contracts", func(t *testing.T) {
		ctx := tests.Context(t)
		tc := newReaderWriterMock(t)
		tcFn := func() (client.ReaderWriter, error) { return tc, nil }
		loopKs := newKeystore(1)
		txm := NewTxm(db, tcFn, *gpe, chainID, cfg, loopKs, lggr)

		id1, err := txm.Enqueue(ctx, contract.String(), generateExecuteMsg([]byte(`0`), sender1, contract))
		require.NoError(t, err)
		id2, err := txm.Enqueue(ctx, contract2.String(), generateExecuteMsg([]byte(`1`), sender2, contract2))
		require.NoError(t, err)
		ids := []int64{id1, id2}
		senders := []string{sender1.String(), sender2.String()}
		contracts := []string{contract.String(), contract2.String()}
		for i := 0; i < 2; i++ {
			tc.On("Account", mock.Anything).Return(uint64(0), uint64(0), nil).Once()
			// Note this must be arg dependent, we don't know which order
			// the procesing will happen in (map iteration by from address).
			tc.On("BatchSimulateUnsigned", client.SimMsgs{
				{
					ID: ids[i],
					Msg: &wasmtypes.MsgExecuteContract{
						Sender:   senders[i],
						Msg:      []byte(fmt.Sprintf(`%d`, i)),
						Contract: contracts[i],
					},
				},
			}, mock.Anything).Return(&client.BatchSimResults{
				Failed: nil,
				Succeeded: client.SimMsgs{
					{
						ID: ids[i],
						Msg: &wasmtypes.MsgExecuteContract{
							Sender:   senders[i],
							Msg:      []byte(fmt.Sprintf(`%d`, i)),
							Contract: contracts[i],
						},
					},
				},
			}, nil).Once()
			tc.On("SimulateUnsigned", mock.Anything, mock.Anything).Return(&txtypes.SimulateResponse{GasInfo: &cosmostypes.GasInfo{
				GasUsed: 1_000_000,
			}}, nil).Once()
			tc.On("LatestBlock").Return(&tmservicetypes.GetLatestBlockResponse{SdkBlock: &tmservicetypes.Block{
				Header: tmservicetypes.Header{Height: 1},
			}}, nil).Once()
			tc.On("CreateAndSign", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]byte{0x01}, nil).Once()
		}
		txResp := &cosmostypes.TxResponse{TxHash: "4BF5122F344554C53BDE2EBB8CD2B7E3D1600AD631C385A5D7CCE23C7785459A"}
		tc.On("Broadcast", mock.Anything, mock.Anything).Return(&txtypes.BroadcastTxResponse{TxResponse: txResp}, nil).Twice()
		tc.On("Tx", mock.Anything).Return(&txtypes.GetTxResponse{Tx: &txtypes.Tx{}, TxResponse: txResp}, nil).Twice()
		txm.sendMsgBatch(tests.Context(t))

		// Should be in completed state
		completed, err := txm.orm.GetMsgs(ctx, id1, id2)
		require.NoError(t, err)
		require.Equal(t, 2, len(completed))
		assert.Equal(t, cosmosdb.Confirmed, completed[0].State)
		assert.Equal(t, cosmosdb.Confirmed, completed[1].State)
	})

	t.Run("failed to confirm", func(t *testing.T) {
		ctx := tests.Context(t)
		tc := newReaderWriterMock(t)
		tc.On("Tx", mock.Anything).Return(&txtypes.GetTxResponse{
			Tx:         &txtypes.Tx{},
			TxResponse: &cosmostypes.TxResponse{TxHash: "0x123"},
		}, errors.New("not found")).Twice()
		tcFn := func() (client.ReaderWriter, error) { return tc, nil }
		loopKs := newKeystore(1)
		txm := NewTxm(db, tcFn, *gpe, chainID, cfg, loopKs, lggr)
		i, err := txm.orm.InsertMsg(ctx, "blah", "", []byte{0x01})
		require.NoError(t, err)
		txh := "0x123"
		require.NoError(t, txm.orm.UpdateMsgs(ctx, []int64{i}, cosmosdb.Started, &txh))
		require.NoError(t, txm.orm.UpdateMsgs(ctx, []int64{i}, cosmosdb.Broadcasted, &txh))
		err = txm.confirmTx(tests.Context(t), tc, txh, []int64{i}, 2, 1*time.Millisecond)
		require.NoError(t, err)
		m, err := txm.orm.GetMsgs(ctx, i)
		require.NoError(t, err)
		require.Equal(t, 1, len(m))
		assert.Equal(t, cosmosdb.Errored, m[0].State)
	})

	t.Run("confirm any unconfirmed", func(t *testing.T) {
		ctx := tests.Context(t)
		require.Equal(t, int64(2), cfg.MaxMsgsPerBatch())
		txHash1 := "0x1234"
		txHash2 := "0x1235"
		txHash3 := "0xabcd"
		tc := newReaderWriterMock(t)
		tc.On("Tx", txHash1).Return(&txtypes.GetTxResponse{
			TxResponse: &cosmostypes.TxResponse{TxHash: txHash1},
		}, nil).Once()
		tc.On("Tx", txHash2).Return(&txtypes.GetTxResponse{
			TxResponse: &cosmostypes.TxResponse{TxHash: txHash2},
		}, nil).Once()
		tc.On("Tx", txHash3).Return(&txtypes.GetTxResponse{
			TxResponse: &cosmostypes.TxResponse{TxHash: txHash3},
		}, nil).Once()
		tcFn := func() (client.ReaderWriter, error) { return tc, nil }
		loopKs := newKeystore(1)
		txm := NewTxm(db, tcFn, *gpe, chainID, cfg, loopKs, lggr)

		// Insert and broadcast 3 msgs with different txhashes.
		id1, err := txm.orm.InsertMsg(ctx, "blah", "", []byte{0x01})
		require.NoError(t, err)
		id2, err := txm.orm.InsertMsg(ctx, "blah", "", []byte{0x02})
		require.NoError(t, err)
		id3, err := txm.orm.InsertMsg(ctx, "blah", "", []byte{0x03})
		require.NoError(t, err)
		err = txm.orm.UpdateMsgs(ctx, []int64{id1}, cosmosdb.Started, &txHash1)
		require.NoError(t, err)
		err = txm.orm.UpdateMsgs(ctx, []int64{id2}, cosmosdb.Started, &txHash2)
		require.NoError(t, err)
		err = txm.orm.UpdateMsgs(ctx, []int64{id3}, cosmosdb.Started, &txHash3)
		require.NoError(t, err)
		err = txm.orm.UpdateMsgs(ctx, []int64{id1}, cosmosdb.Broadcasted, &txHash1)
		require.NoError(t, err)
		err = txm.orm.UpdateMsgs(ctx, []int64{id2}, cosmosdb.Broadcasted, &txHash2)
		require.NoError(t, err)
		err = txm.orm.UpdateMsgs(ctx, []int64{id3}, cosmosdb.Broadcasted, &txHash3)
		require.NoError(t, err)

		// Confirm them as in a restart while confirming scenario
		txm.confirmAnyUnconfirmed(ctx)
		msgs, err := txm.orm.GetMsgs(ctx, id1, id2, id3)
		require.NoError(t, err)
		require.Equal(t, 3, len(msgs))
		assert.Equal(t, cosmosdb.Confirmed, msgs[0].State)
		assert.Equal(t, cosmosdb.Confirmed, msgs[1].State)
		assert.Equal(t, cosmosdb.Confirmed, msgs[2].State)
	})

	t.Run("expired msgs", func(t *testing.T) {
		ctx := tests.Context(t)
		tc := new(mocks.ReaderWriter)
		timeout, err := utils.NewDuration(1 * time.Millisecond)
		require.NoError(t, err)
		tcFn := func() (client.ReaderWriter, error) { return tc, nil }
		two := int64(2)
		cfgShortExpiry := &config.TOMLConfig{Chain: config.Chain{
			MaxMsgsPerBatch: &two,
			TxMsgTimeout:    &timeout,
		}}
		cfgShortExpiry.SetDefaults()
		loopKs := newKeystore(1)
		txm := NewTxm(db, tcFn, *gpe, chainID, cfgShortExpiry, loopKs, lggr)

		// Send a single one expired
		id1, err := txm.orm.InsertMsg(ctx, "blah", "", []byte{0x03})
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
		txm.sendMsgBatch(tests.Context(t))
		// Should be marked errored
		m, err := txm.orm.GetMsgs(ctx, id1)
		require.NoError(t, err)
		assert.Equal(t, cosmosdb.Errored, m[0].State)

		// Send a batch which is all expired
		id2, err := txm.orm.InsertMsg(ctx, "blah", "", []byte{0x03})
		require.NoError(t, err)
		id3, err := txm.orm.InsertMsg(ctx, "blah", "", []byte{0x03})
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
		txm.sendMsgBatch(tests.Context(t))
		require.NoError(t, err)
		ms, err := txm.orm.GetMsgs(ctx, id2, id3)
		require.NoError(t, err)
		assert.Equal(t, cosmosdb.Errored, ms[0].State)
		assert.Equal(t, cosmosdb.Errored, ms[1].State)
	})

	t.Run("started msgs", func(t *testing.T) {
		ctx := tests.Context(t)
		tc := new(mocks.ReaderWriter)
		tc.On("Account", mock.Anything).Return(uint64(0), uint64(0), nil)
		tc.On("SimulateUnsigned", mock.Anything, mock.Anything).Return(&txtypes.SimulateResponse{GasInfo: &cosmostypes.GasInfo{
			GasUsed: 1_000_000,
		}}, nil)
		tc.On("LatestBlock").Return(&tmservicetypes.GetLatestBlockResponse{SdkBlock: &tmservicetypes.Block{
			Header: tmservicetypes.Header{Height: 1},
		}}, nil)
		tc.On("CreateAndSign", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]byte{0x01}, nil)
		txResp := &cosmostypes.TxResponse{TxHash: "4BF5122F344554C53BDE2EBB8CD2B7E3D1600AD631C385A5D7CCE23C7785459A"}
		tc.On("Broadcast", mock.Anything, mock.Anything).Return(&txtypes.BroadcastTxResponse{TxResponse: txResp}, nil)
		tc.On("Tx", mock.Anything).Return(&txtypes.GetTxResponse{Tx: &txtypes.Tx{}, TxResponse: txResp}, nil)
		tcFn := func() (client.ReaderWriter, error) { return tc, nil }
		two := int64(2)
		cfgMaxMsgs := &config.TOMLConfig{Chain: config.Chain{
			MaxMsgsPerBatch: &two,
		}}
		cfgMaxMsgs.SetDefaults()
		loopKs := newKeystore(1)
		txm := NewTxm(db, tcFn, *gpe, chainID, cfgMaxMsgs, loopKs, lggr)

		// Leftover started is processed
		msg1 := generateExecuteMsg([]byte{0x03}, sender1, contract)
		id1 := mustInsertMsg(t, txm, contract.String(), msg1)
		require.NoError(t, txm.orm.UpdateMsgs(ctx, []int64{id1}, cosmosdb.Started, nil))
		msgs := client.SimMsgs{{ID: id1, Msg: &wasmtypes.MsgExecuteContract{
			Sender:   sender1.String(),
			Msg:      []byte{0x03},
			Contract: contract.String(),
		}}}
		tc.On("BatchSimulateUnsigned", msgs, mock.Anything).
			Return(&client.BatchSimResults{Failed: nil, Succeeded: msgs}, nil).Once()
		time.Sleep(1 * time.Millisecond)
		txm.sendMsgBatch(tests.Context(t))
		m, err := txm.orm.GetMsgs(ctx, id1)
		require.NoError(t, err)
		assert.Equal(t, cosmosdb.Confirmed, m[0].State)

		// Leftover started is not cancelled
		msg2 := generateExecuteMsg([]byte{0x04}, sender1, contract)
		msg3 := generateExecuteMsg([]byte{0x05}, sender1, contract)
		id2 := mustInsertMsg(t, txm, contract.String(), msg2)
		require.NoError(t, txm.orm.UpdateMsgs(ctx, []int64{id2}, cosmosdb.Started, nil))
		time.Sleep(time.Millisecond) // ensure != CreatedAt
		id3 := mustInsertMsg(t, txm, contract.String(), msg3)
		msgs = client.SimMsgs{{ID: id2, Msg: &wasmtypes.MsgExecuteContract{
			Sender:   sender1.String(),
			Msg:      []byte{0x04},
			Contract: contract.String(),
		}}, {ID: id3, Msg: &wasmtypes.MsgExecuteContract{
			Sender:   sender1.String(),
			Msg:      []byte{0x05},
			Contract: contract.String(),
		}}}
		tc.On("BatchSimulateUnsigned", msgs, mock.Anything).
			Return(&client.BatchSimResults{Failed: nil, Succeeded: msgs}, nil).Once()
		time.Sleep(1 * time.Millisecond)
		txm.sendMsgBatch(tests.Context(t))
		require.NoError(t, err)
		ms, err := txm.orm.GetMsgs(ctx, id2, id3)
		require.NoError(t, err)
		assert.Equal(t, cosmosdb.Confirmed, ms[0].State)
		assert.Equal(t, cosmosdb.Confirmed, ms[1].State)
	})
}

func mustInsertMsg(t *testing.T, txm *Txm, contractID string, msg cosmostypes.Msg) int64 {
	typeURL, raw, err := txm.marshalMsg(msg)
	require.NoError(t, err)
	id, err := txm.orm.InsertMsg(tests.Context(t), contractID, typeURL, raw)
	require.NoError(t, err)
	return id
}

// RandomChainID returns a random chain id for testing. Use this instead of a constant to prevent DB collisions.
func RandomChainID() string {
	return fmt.Sprintf("Chainlinktest-%s", uuid.New())
}

type keystore struct {
	accounts []string
}

func newKeystore(count int) *keystore {
	accounts := make([]string, count)
	for i := 0; i < count; i++ {
		accounts[i] = cosmostypes.AccAddress(secp256k1.GenPrivKey().PubKey().Address().Bytes()).String()
	}
	return &keystore{accounts}
}

func (k *keystore) Accounts(ctx context.Context) (accounts []string, err error) {
	return k.accounts, nil
}

func (k *keystore) Sign(ctx context.Context, account string, data []byte) (signed []byte, err error) {
	if slices.Index(k.accounts, account) == -1 {
		return nil, fmt.Errorf("account not found: %s", account)
	}
	return data, nil
}
