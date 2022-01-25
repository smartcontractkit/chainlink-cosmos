package terra

import (
	"context"
	"encoding/json"
	cosmosSDK "github.com/cosmos/cosmos-sdk/types"
	terraSDK "github.com/terra-money/core/x/wasm/types"

	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

var _ types.ContractTransmitter = (*ContractTransmitter)(nil)

type ContractTransmitter struct {
	msgEnqueuer MsgEnqueuer
	lggr        Logger
	jobID       string
	contract    cosmosSDK.AccAddress
	sender      cosmosSDK.AccAddress
	cfg         Config
	reader      *OCR2Reader
}

func NewContractTransmitter(jobID string,
	contract cosmosSDK.AccAddress,
	sender cosmosSDK.AccAddress,
	msgEnqueuer MsgEnqueuer,
	lggr Logger,
	cfg Config,
	reader *OCR2Reader,
) *ContractTransmitter {
	return &ContractTransmitter{
		jobID:       jobID,
		contract:    contract,
		msgEnqueuer: msgEnqueuer,
		sender:      sender,
		lggr:        lggr,
		cfg:         cfg,
		reader:      reader,
	}
}

// Transmit signs and sends the report
func (ct *ContractTransmitter) Transmit(
	ctx context.Context,
	reportCtx types.ReportContext,
	report types.Report,
	sigs []types.AttributedOnchainSignature,
) error {
	ct.lggr.Infof("[%s] Sending TX to %s", ct.jobID, ct.contract.String())
	msgStruct := TransmitMsg{}
	reportContext := evmutil.RawReportContext(reportCtx)
	for _, r := range reportContext {
		msgStruct.Transmit.ReportContext = append(msgStruct.Transmit.ReportContext, r[:]...)
	}
	msgStruct.Transmit.Report = []byte(report)
	for _, sig := range sigs {
		msgStruct.Transmit.Signatures = append(msgStruct.Transmit.Signatures, sig.Signature)
	}
	msgBytes, err := json.Marshal(msgStruct)
	if err != nil {
		return err
	}
	m := terraSDK.NewMsgExecuteContract(ct.sender, ct.contract, msgBytes, cosmosSDK.Coins{})
	d, err := m.Marshal()
	if err != nil {
		return err
	}
	_, err = ct.msgEnqueuer.Enqueue(ct.contract.String(), d)
	return err
}

// LatestConfigDigestAndEpoch fetches the latest details from cache
func (ct *ContractTransmitter) LatestConfigDigestAndEpoch(ctx context.Context) (
	configDigest types.ConfigDigest,
	epoch uint32,
	err error,
) {
	return ct.reader.latestConfigDigestAndEpoch(ctx)
}

func (ct *ContractTransmitter) FromAccount() types.Account {
	return types.Account(ct.sender.String())
}
