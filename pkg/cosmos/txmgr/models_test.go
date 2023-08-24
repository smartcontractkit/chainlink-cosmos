package txmgr

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/assert"
)

func TestEncodeDecodePayload(t *testing.T) {
	fromAddress, err := sdk.AccAddressFromBech32("cosmos1swp4kg5e3vphyufvllenu4uet6368f88cwcmc0")
	assert.NoError(t, err)
	toAddress, err := sdk.AccAddressFromBech32("cosmos153lf4zntqt33a4v0sm5cytrxyqn78q7kz8j8x5")
	assert.NoError(t, err)

	msgSend := types.MsgSend{
		FromAddress: fromAddress.String(),
		ToAddress:   toAddress.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(1000))),
	}

	encoded, err := EncodePayload(&msgSend)
	assert.NoError(t, err)

	decoded, err := DecodePayload(typeMsgSend, encoded)
	assert.NoError(t, err)

	assert.Equal(t, &msgSend, decoded)
}

func TestAddress(t *testing.T) {
	addr32, err := sdk.AccAddressFromHexUnsafe("b9df9bb7c25b05196fc5a4237eedab76460347da45b7b6bbe5edfa5d396c42a6")
	assert.NoError(t, err)
	assert.Equal(t, 32, len(addr32.Bytes()))

	addr20, err := sdk.AccAddressFromBech32("cosmos1swp4kg5e3vphyufvllenu4uet6368f88cwcmc0")
	assert.NoError(t, err)
	assert.Equal(t, 20, len(addr20.Bytes()))

	testCases := []struct {
		name       string
		accAddress sdk.AccAddress
	}{
		{
			name:       "32_bytes_address",
			accAddress: addr32,
		},
		{
			name:       "20_bytes_address",
			accAddress: addr20,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_NewAddress", func(t *testing.T) {
			addr := NewAddress(tc.accAddress)
			if !bytes.Equal(addr.Bytes()[:len(tc.accAddress)], tc.accAddress) {
				t.Errorf("NewAddress() = %v, want %v", addr.Bytes(), tc.accAddress)
			}
		})

		t.Run(tc.name+"_ToCosmosAddress", func(t *testing.T) {
			addr := NewAddress(tc.accAddress)
			cosmosAddr := ToCosmosAddress(addr)
			if cosmosAddr.String() != tc.accAddress.String() {
				t.Errorf("ToCosmosAddress() = %v, want %v", cosmosAddr, tc.accAddress)
			}
		})

		t.Run(tc.name+"_String", func(t *testing.T) {
			addr := NewAddress(tc.accAddress)
			if addr.String() != tc.accAddress.String() {
				t.Errorf("String() = %v, want %v", addr.String(), tc.accAddress.String())
			}
		})
	}
}
