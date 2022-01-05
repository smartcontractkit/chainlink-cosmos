package terra

import (
	"context"
	"encoding/json"
	cosmosSDK "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	terraSDK "github.com/terra-money/core/x/wasm/types"
)

var _ types.ContractTransmitter = (*ContractTransmitter)(nil)

type ContractTransmitter struct {
	transmissionSigner TransmissionSigner
	client             TerraReaderWriter
	lggr               Logger
	jobID              string
	contract           cosmosSDK.AccAddress
}

func NewContractTransmitter(jobID string,
	contract cosmosSDK.AccAddress,
	ts TransmissionSigner,
	client TerraReaderWriter,
	lggr Logger,
) *ContractTransmitter {
	return &ContractTransmitter{
		jobID:              jobID,
		contract:           contract,
		transmissionSigner: ts,
		client:             client,
		lggr:               lggr,
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
	sender, err := cosmosSDK.AccAddressFromBech32(ct.transmissionSigner.PublicKey().String())
	if err != nil {
		return err
	}
	msg := terraSDK.NewMsgExecuteContract(sender, ct.contract, msgBytes, cosmosSDK.Coins{})
	sn, err := ct.client.SequenceNumber(sender)
	if err != nil {
		return err
	}
	txResponse, err := ct.client.SignAndBroadcast(msg, sn, ct.client.GasPrice(), WrappedPrivKey{ct.transmissionSigner})
	if err != nil {
		return errors.Wrap(err, "error in Transmit.Send")
	}
	ct.lggr.Infof("[%s] TX Hash: %s", ct.jobID, txResponse.TxHash)
	return nil
}

// LatestConfigDigestAndEpoch fetches the latest details from contract state
func (ct *ContractTransmitter) LatestConfigDigestAndEpoch(ctx context.Context) (
	configDigest types.ConfigDigest,
	epoch uint32,
	err error,
) {
	resp, err := ct.client.QueryABCI(
		"custom/wasm/contractStore",
		NewAbciQueryParams(ct.contract.String(), []byte(`"latest_config_digest_and_epoch"`)),
	)
	if err != nil {
		return types.ConfigDigest{}, 0, err
	}

	var digest LatestConfigDigestAndEpoch
	if err := json.Unmarshal(resp.Value, &digest); err != nil {
		return types.ConfigDigest{}, 0, err
	}

	return digest.ConfigDigest, digest.Epoch, nil
}

func (ct *ContractTransmitter) FromAccount() types.Account {
	return types.Account(ct.transmissionSigner.PublicKey().String())
}
