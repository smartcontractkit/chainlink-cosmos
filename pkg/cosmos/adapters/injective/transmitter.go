package injective

import (
	"context"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/injective/median_report"
	chaintypes "github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/injective/types"
)

var _ types.ContractTransmitter = &CosmosModuleTransmitter{}

type CosmosModuleTransmitter struct {
	lggr        logger.Logger
	FeedId      string
	QueryClient chaintypes.QueryClient
	ReportCodec median_report.ReportCodec
	msgEnqueuer cosmos.MsgEnqueuer
	contract    cosmosSDK.AccAddress
	sender      cosmosSDK.AccAddress
}

func (c *CosmosModuleTransmitter) FromAccount() types.Account {
	return types.Account(c.sender.String())
}

// Transmit sends the report to the on-chain OCR2Aggregator smart contract's Transmit method
func (c *CosmosModuleTransmitter) Transmit(
	ctx context.Context,
	reportCtx types.ReportContext,
	report types.Report,
	signatures []types.AttributedOnchainSignature,
) error {
	if len(c.FeedId) == 0 {
		err := errors.New("CosmosModuleTransmitter has no FeedId set")
		return err
	}

	// TODO: design how to decouple Cosmos reporting from reportingplugins of OCR2
	// The reports are not necessarily numeric (see: titlerequest).
	reportRaw, err := c.ReportCodec.ParseReport(report)
	if err != nil {
		return err
	}

	msgTransmit := &chaintypes.MsgTransmit{
		Transmitter:  c.sender.String(),
		ConfigDigest: reportCtx.ConfigDigest[:],
		FeedId:       c.FeedId,
		Epoch:        uint64(reportCtx.Epoch),
		Round:        uint64(reportCtx.Round),
		ExtraHash:    reportCtx.ExtraHash[:],
		Report: &chaintypes.Report{ // chain only understands median.Report for now
			ObservationsTimestamp: reportRaw.ObservationsTimestamp,
			Observers:             reportRaw.Observers,
			Observations:          reportRaw.Observations,
		},
		Signatures: make([][]byte, 0, len(signatures)),
	}

	for _, sig := range signatures {
		msgTransmit.Signatures = append(msgTransmit.Signatures, sig.Signature)
	}

	_, err = c.msgEnqueuer.Enqueue(c.contract.String(), msgTransmit)
	return err
}

func (c *CosmosModuleTransmitter) LatestConfigDigestAndEpoch(
	ctx context.Context,
) (
	configDigest types.ConfigDigest,
	epoch uint32,
	err error,
) {
	if len(c.FeedId) == 0 {
		err := errors.New("CosmosModuleTransmitter has no FeedId set")
		return types.ConfigDigest{}, 0, err
	}

	if c.QueryClient == nil {
		err := errors.New("cannot query LatestConfigDigestAndEpoch: no QueryClient set")
		return types.ConfigDigest{}, 0, err
	}

	resp, err := c.QueryClient.FeedConfigInfo(ctx, &chaintypes.QueryFeedConfigInfoRequest{
		FeedId: c.FeedId,
	})
	if err != nil {
		return types.ConfigDigest{}, 0, err
	}

	configDigest = configDigestFromBytes(resp.FeedConfigInfo.LatestConfigDigest)
	epoch = uint32(resp.EpochAndRound.Epoch)
	return configDigest, epoch, nil
}
