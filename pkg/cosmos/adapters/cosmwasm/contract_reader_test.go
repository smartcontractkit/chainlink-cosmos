package cosmwasm

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

func Test_parseAttributes(t *testing.T) {
	valid := []cosmosSDK.Attribute{
		{Key: "config_count", Value: "1"},
		{Key: "f", Value: "79"},
		{Key: "latest_config_digest", Value: "7465737420636f6e66696720646967657374203332206368617273206c6f6e67"},
		{Key: "offchain_config", Value: "AwQ="},
		{Key: "offchain_config_version", Value: "111"},
		{Key: "onchain_config", Value: "AQI="},
		{Key: "signers", Value: "0101010101010101010101010101010101010101010101010101010101010101"},
		{Key: "signers", Value: "0202020202020202020202020202020202020202020202020202020202020202"},
		{Key: "transmitters", Value: "account1"},
		{Key: "transmitters", Value: "account2"},
	}
	validResult := types.ContractConfig{
		ConfigDigest: mustStringToConfigDigest(t, "test config digest 32 chars long"),
		ConfigCount:  1,
		Signers: []types.OnchainPublicKey{
			types.OnchainPublicKey(bytes.Repeat([]byte{0x01}, 32)),
			types.OnchainPublicKey(bytes.Repeat([]byte{0x02}, 32)),
		},
		Transmitters:          []types.Account{"account1", "account2"},
		F:                     79,
		OnchainConfig:         []byte{0x01, 0x02},
		OffchainConfigVersion: 111,
		OffchainConfig:        []byte{0x03, 0x04},
	}
	tests := []struct {
		name       string
		attrs      []cosmosSDK.Attribute
		exp        types.ContractConfig
		expErrIs   error
		expErrStr  string
		expUnknown map[string]int
	}{
		{name: "valid", attrs: valid, exp: validResult},
		{
			name:       "valid-unknown",
			attrs:      append(valid, cosmosSDK.Attribute{Key: "foo"}, cosmosSDK.Attribute{Key: "foo"}, cosmosSDK.Attribute{Key: "bar"}),
			exp:        validResult,
			expUnknown: map[string]int{"foo": 2, "bar": 1},
		},

		// invalid
		{name: "empty", attrs: nil, exp: types.ContractConfig{}, expErrStr: "expected 8 types of known keys"},
		{name: "missing_config-count", attrs: valid[1:], expErrStr: "expected 8 types of known keys"},
		{name: "dupe_config-count", attrs: append([]cosmosSDK.Attribute{valid[0]}, valid...), expErrIs: ErrAttrDupe("config_count")},
		{name: "config_count-decimal", expErrIs: strconv.ErrSyntax, attrs: []cosmosSDK.Attribute{
			{Key: "config_count", Value: "1.1"}}},
		{name: "f-hex", expErrIs: strconv.ErrSyntax, attrs: []cosmosSDK.Attribute{
			{Key: "f", Value: "0xabcd"}}},
		{name: "latest_config_digest-truncated", expErrStr: "cannot convert bytes to ConfigDigest. bytes have wrong length", attrs: []cosmosSDK.Attribute{
			{Key: "latest_config_digest", Value: "7465737420636f6e6669672064"}}},
		{name: "offchain_config-hex", expErrIs: base64.CorruptInputError(4), attrs: []cosmosSDK.Attribute{
			{Key: "offchain_config", Value: "0x1234"}}},
		{name: "offchain_config_version-word", expErrIs: strconv.ErrSyntax, attrs: []cosmosSDK.Attribute{
			{Key: "offchain_config_version", Value: "hundred"}}},
		{name: "signers-base64", expErrIs: hex.InvalidByteError('Q'), attrs: []cosmosSDK.Attribute{
			{Key: "signers", Value: "AQ=="}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, unknown, err := parseAttributes(tt.attrs)
			if tt.expErrIs != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expErrIs)
			} else if tt.expErrStr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expErrStr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.exp, got)
			}
			require.Equal(t, tt.expUnknown, unknown)
		})
	}
}

func mustStringToConfigDigest(t *testing.T, s string) types.ConfigDigest {
	d, err := types.BytesToConfigDigest([]byte(s))
	require.NoError(t, err)
	return d
}
