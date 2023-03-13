package injective

import (
	"context"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/injective/median_report"
	chaintypes "github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/injective/types"
)

var _ types.ContractTransmitter = &CosmosModuleTransmitter{}

type CosmosModuleTransmitter struct {
	lggr        logger.Logger
	queryClient chaintypes.QueryClient
	msgEnqueuer adapters.MsgEnqueuer
	feedID      string
	sender      cosmosSDK.AccAddress
}

func NewCosmosModuleTransmitter(
	queryClient chaintypes.QueryClient,
	feedId string,
	sender cosmosSDK.AccAddress,
	msgEnqueuer adapters.MsgEnqueuer,
	lggr logger.Logger,
) *CosmosModuleTransmitter {
	return &CosmosModuleTransmitter{
		lggr:        lggr,
		feedID:      feedId,
		queryClient: queryClient,
		msgEnqueuer: msgEnqueuer,
		sender:      sender,
	}
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
	// TODO: design how to decouple Cosmos reporting from reportingplugins of OCR2
	// The reports are not necessarily numeric (see: titlerequest).
	reportRaw, err := median_report.ParseReport(report)
	if err != nil {
		return err
	}

	msgTransmit := &chaintypes.MsgTransmit{
		Transmitter:  c.sender.String(),
		ConfigDigest: reportCtx.ConfigDigest[:],
		FeedId:       c.feedID,
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

	_, err = c.msgEnqueuer.Enqueue(c.feedID, msgTransmit)
	return err
}

func (c *CosmosModuleTransmitter) LatestConfigDigestAndEpoch(
	ctx context.Context,
) (
	configDigest types.ConfigDigest,
	epoch uint32,
	err error,
) {
	resp, err := c.queryClient.FeedConfigInfo(ctx, &chaintypes.QueryFeedConfigInfoRequest{
		FeedId: c.feedID,
	})
	if err != nil {
		return types.ConfigDigest{}, 0, err
	}

	configDigest = configDigestFromBytes(resp.FeedConfigInfo.LatestConfigDigest)
	epoch = uint32(resp.EpochAndRound.Epoch)
	return configDigest, epoch, nil
}
