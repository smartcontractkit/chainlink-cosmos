package injective

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	chaintypes "github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/injective/types"
)

const ConfigDigestPrefixCosmos types.ConfigDigestPrefix = 2

var _ types.OffchainConfigDigester = CosmosOffchainConfigDigester{}

type CosmosOffchainConfigDigester struct {
	ChainID string
	FeedID  string
}

func (d CosmosOffchainConfigDigester) ConfigDigest(cc types.ContractConfig) (types.ConfigDigest, error) {
	signers := make([]string, 0, len(cc.Signers))
	for _, acc := range cc.Signers {
		signers = append(signers, sdk.AccAddress(acc).String())
	}

	transmitters := make([]string, 0, len(cc.Transmitters))
	for _, acc := range cc.Transmitters {
		addr, err := sdk.AccAddressFromBech32(string(acc))
		if err != nil {
			return types.ConfigDigest{}, err
		}

		transmitters = append(transmitters, addr.String())
	}

	chainContractConfig := &chaintypes.ContractConfig{
		ConfigCount:           cc.ConfigCount,
		Signers:               signers,
		Transmitters:          transmitters,
		F:                     uint32(cc.F),
		OnchainConfig:         cc.OnchainConfig,
		OffchainConfigVersion: cc.OffchainConfigVersion,
		OffchainConfig:        cc.OffchainConfig,
	}

	configDigest := configDigestFromBytes(chainContractConfig.Digest(d.ChainID, d.FeedID))

	return configDigest, nil
}

func (d CosmosOffchainConfigDigester) ConfigDigestPrefix() types.ConfigDigestPrefix {
	return ConfigDigestPrefixCosmos
}

func configDigestFromBytes(buf []byte) types.ConfigDigest {
	var configDigest types.ConfigDigest

	if len(buf) != len(configDigest) {
		// assertion
		panic("buffer is not matching digest/hash length (32)")
	}

	if n := copy(configDigest[:], buf); n != len(configDigest) {
		// assertion
		panic("unexpectedly short read")
	}

	if configDigest[0] != 0 || types.ConfigDigestPrefix(configDigest[1]) != ConfigDigestPrefixCosmos {
		// assertion
		panic("wrong ConfigDigestPrefix")
	}

	return configDigest
}
