package monitoring

import (
	"context"
	"fmt"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	pkgTerra "github.com/smartcontractkit/chainlink-terra/pkg/terra"
	pkgClient "github.com/smartcontractkit/chainlink-terra/pkg/terra/client"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

func NewTerraSourceFactory(log pkgTerra.Logger) relayMonitoring.SourceFactory {
	return &sourceFactory{log}
}

type sourceFactory struct {
	log pkgTerra.Logger
}

func (s *sourceFactory) NewSource(
	chainConfig relayMonitoring.ChainConfig,
	feedConfig relayMonitoring.FeedConfig,
) (relayMonitoring.Source, error) {
	terraChainConfig, ok := chainConfig.(TerraConfig)
	if !ok {
		return nil, fmt.Errorf("expected chainConfig to be of type TerraConfig not %T", chainConfig)
	}
	terraFeedConfig, ok := feedConfig.(TerraFeedConfig)
	if !ok {
		return nil, fmt.Errorf("expected feedConfig to be of type TerraFeedConfig not %T", feedConfig)
	}
	client, err := pkgClient.NewClient(
		terraChainConfig.TendermintURL,
		terraChainConfig.FCDURL,
		terraChainConfig.ReadTimeout,
		s.log,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create a new terra client: %w", err)
	}
	medianContract := pkgTerra.NewMedianContract(
		terraFeedConfig.ContractAddress,
		client,
		s.log,
		nil, // transmitter is not needed because we don't read the chain.
		nil, // config is also not needed for reading the chain.
	)
	contractTracker := pkgTerra.NewContractTracker(
		terraFeedConfig.ContractAddress,
		"fake-job-id", //jobID
		client,
		s.log,
	)
	return &terraSource{
		medianContract,
		contractTracker,
	}, nil
}

type terraSource struct {
	medianContract  *pkgTerra.MedianContract
	contractTracker *pkgTerra.ContractTracker
}

func (s *terraSource) Fetch(ctx context.Context) (interface{}, error) {
	changedInBlock, _, err := s.contractTracker.LatestConfigDetails(ctx)
	if err != nil {
		return relayMonitoring.Envelope{}, fmt.Errorf("failed to fetch latest config details from on-chain: %w", err)
	}
	cfg, err := s.contractTracker.LatestConfig(ctx, changedInBlock)
	if err != nil {
		return relayMonitoring.Envelope{}, fmt.Errorf("failed to read latest config from on-chain: %w", err)
	}
	configDigest, epoch, round, latestAnswer, latestTimestamp, err := s.medianContract.LatestTransmissionDetails(ctx)
	if err != nil {
		return relayMonitoring.Envelope{}, fmt.Errorf("failed to read latest transmission from on-chain: %w", err)
	}
	transmitter := types.Account("test")

	return relayMonitoring.Envelope{
		configDigest,
		epoch,
		round,
		latestAnswer,
		latestTimestamp,

		cfg,

		changedInBlock,
		transmitter,
	}, nil
}
