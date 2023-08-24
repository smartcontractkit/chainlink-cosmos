package txmgr

import (
	"bytes"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos"
	txmgrtypes "github.com/smartcontractkit/chainlink-relay/pkg/txmgr/types"
)

// Type aliases for Cosmos specific implementation of generic txmgr
type (
	TxRequest = txmgrtypes.TxRequest[Address, TxHash]
)

var (
	typeMsgSend            = sdk.MsgTypeURL(&types.MsgSend{})
	typeMsgExecuteContract = sdk.MsgTypeURL(&wasmtypes.MsgExecuteContract{})
)

// Encodes cosmos native txn type into bytes for generic txmgr
func EncodePayload(msg sdk.Msg) ([]byte, error) {
	switch ms := msg.(type) {
	case *wasmtypes.MsgExecuteContract:
		_, err := sdk.AccAddressFromBech32(ms.Sender)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to encode payload as parse sender failed: %s", ms.Sender)
		}

	case *types.MsgSend:
		_, err := sdk.AccAddressFromBech32(ms.FromAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to encode payload as parse sender failed: %s", ms.FromAddress)
		}

	default:
		return nil, &cosmos.ErrMsgUnsupported{Msg: msg}
	}
	raw, err := proto.Marshal(msg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to encode payload as proto marshal failed: %s", msg)
	}
	return raw, nil
}

// Decodes bytes into cosmos native txn type
func DecodePayload(msgType string, raw []byte) (msg sdk.Msg, err error) {
	switch msgType {
	case typeMsgSend:
		var ms types.MsgSend
		err := ms.Unmarshal(raw)
		if err != nil {
			return nil, err
		}
		return &ms, nil
	case typeMsgExecuteContract:
		var ms wasmtypes.MsgExecuteContract
		err := ms.Unmarshal(raw)
		if err != nil {
			return nil, err
		}
		return &ms, nil
	}
	return nil, errors.Errorf("unrecognized message type: %s", msgType)
}

// A wrapper for sdk.Address. Uses fixed max address size of 32-byte size to satisfy Hashable interface.
// Cosmos addresses shorter than 32 bytes are zero-padded.
// https://docs.cosmos.network/v0.46/basics/accounts.html#keys-accounts-addresses-and-signatures
type Address [32]byte

// Create a new Address from a sdk.Address
func NewAddress(addr sdk.AccAddress) Address {
	var a Address
	copy(a[:], addr[:])
	return a
}

func ToCosmosAddress(addr Address) sdk.AccAddress {
	return sdk.AccAddress(bytes.TrimRight(addr[:], "\x00"))
}

func (a Address) Bytes() []byte {
	return a[:]
}

func (a Address) String() string {
	return ToCosmosAddress(a).String()
}

// Wrapper type for tx hash in generic txmgr
type TxHash string

func (h TxHash) Bytes() []byte {
	return []byte(h)
}

func (h TxHash) String() string {
	return string(h)
}
