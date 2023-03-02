package injective

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	tmtypes "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	chaintypes "github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/injective/types"
)

var _ types.ContractConfigTracker = &CosmosModuleConfigTracker{}

type CosmosModuleConfigTracker struct {
	FeedId                  string
	QueryClient             chaintypes.QueryClient
	tendermintServiceClient tmtypes.ServiceClient
}

// Notify may optionally emit notification events when the contract's
// configuration changes. This is purely used as an optimization reducing
// the delay between a configuration change and its enactment. Implementors
// who don't care about this may simply return a nil channel.
//
// The returned channel should never be closed.
func (c *CosmosModuleConfigTracker) Notify() <-chan struct{} {
	// TODO: track events from Tendermint WS
	return nil
}

// LatestConfigDetails returns information about the latest configuration,
// but not the configuration itself.
func (c *CosmosModuleConfigTracker) LatestConfigDetails(
	ctx context.Context,
) (
	changedInBlock uint64,
	configDigest types.ConfigDigest,
	err error,
) {
	if len(c.FeedId) == 0 {
		err := errors.New("CosmosModuleConfigTracker has no FeedId set")
		return 0, types.ConfigDigest{}, err
	}

	if c.QueryClient == nil {
		err := errors.New("cannot query LatestConfigDetails: no QueryClient set")
		return 0, types.ConfigDigest{}, err
	}

	resp, err := c.QueryClient.FeedConfigInfo(ctx, &chaintypes.QueryFeedConfigInfoRequest{
		FeedId: c.FeedId,
	})
	if err != nil {
		return 0, types.ConfigDigest{}, err
	}

	if resp.FeedConfigInfo == nil {
		err = errors.Errorf("feed config not found: %s", c.FeedId)
		return 0, types.ConfigDigest{}, err
	}

	changedInBlock = uint64(resp.FeedConfigInfo.LatestConfigBlockNumber)
	configDigest = configDigestFromBytes(resp.FeedConfigInfo.LatestConfigDigest)
	return changedInBlock, configDigest, nil
}

// LatestConfig returns the latest configuration.
func (c *CosmosModuleConfigTracker) LatestConfig(
	ctx context.Context,
	changedInBlock uint64,
) (types.ContractConfig, error) {
	if len(c.FeedId) == 0 {
		err := errors.New("CosmosModuleConfigTracker has no FeedId set")
		return types.ContractConfig{}, err
	}

	if c.QueryClient == nil {
		err := errors.New("cannot query LatestConfig: no QueryClient set")
		return types.ContractConfig{}, err
	}

	resp, err := c.QueryClient.FeedConfig(ctx, &chaintypes.QueryFeedConfigRequest{
		FeedId: c.FeedId,
	})
	if err != nil {
		return types.ContractConfig{}, err
	}

	signers := make([]types.OnchainPublicKey, 0, len(resp.FeedConfig.Signers))
	for _, addr := range resp.FeedConfig.Signers {
		acc, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			panic(err)
		}

		signers = append(signers, types.OnchainPublicKey(acc.Bytes()))
	}

	transmitters := make([]types.Account, 0, len(resp.FeedConfig.Transmitters))
	for _, addr := range resp.FeedConfig.Transmitters {
		acc, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			panic(err)
		}

		transmitters = append(transmitters, types.Account(acc.String()))
	}

	config := types.ContractConfig{
		ConfigDigest:          configDigestFromBytes(resp.FeedConfigInfo.LatestConfigDigest),
		ConfigCount:           uint64(resp.FeedConfigInfo.ConfigCount),
		Signers:               signers,
		Transmitters:          transmitters,
		F:                     uint8(resp.FeedConfig.F),
		OnchainConfig:         resp.FeedConfig.OnchainConfig,
		OffchainConfigVersion: resp.FeedConfig.OffchainConfigVersion,
		OffchainConfig:        resp.FeedConfig.OffchainConfig,
	}

	return config, nil
}

// TODO: duplicated from wasm adapter
// LatestBlockHeight returns the height of the most recent block in the chain.
func (c *CosmosModuleConfigTracker) LatestBlockHeight(
	ctx context.Context,
) (
	blockHeight uint64,
	err error,
) {
	b, err := c.tendermintServiceClient.GetLatestBlock(context.Background(), &tmtypes.GetLatestBlockRequest{})
	if err != nil {
		return 0, err
	}
	return uint64(b.Block.Header.Height), nil
}
