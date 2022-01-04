package terra

import (
	"context"
	"encoding/json"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/terra-money/terra.go/client"
	"github.com/terra-money/terra.go/msg"

	cosmosSDK "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	terraSDK "github.com/terra-money/core/x/wasm/types"
)

// Transmit signs and sends the report
func (ct *Contract) Transmit(
	ctx context.Context,
	reportCtx types.ReportContext,
	report types.Report,
	sigs []types.AttributedOnchainSignature,
) error {
	ct.terra.Log.Infof("[%s] Sending TX to %s", ct.JobID, ct.ContractID)
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

	// convert addresses from string to proper types
	sender, err := cosmosSDK.AccAddressFromBech32(ct.Transmitter.PublicKey().String())
	if err != nil {
		return err
	}

	// create execute msg
	rawMsg := terraSDK.NewMsgExecuteContract(sender, ct.ContractID, msgBytes, cosmosSDK.Coins{})
	options := client.CreateTxOptions{
		Msgs: []msg.Msg{rawMsg},
		Memo: "",
	}

	// need LCD for fetching sequence, account number, + setting gas prices, etc
	lcd := ct.terra.LCD(ct.terra.GasPrice(), ct.terra.gasLimitMultiplier, WrappedPrivKey{ct.Transmitter}, ct.terra.httpClient.Timeout)
	txBuilder, err := lcd.CreateAndSignTx(context.TODO(), options)
	if err != nil {
		return errors.Wrap(err, "error in Transmit.NewTxBuilder")
	}
	txBytes, err := txBuilder.GetTxBytes()
	if err != nil {
		return errors.Wrap(err, "error in Transmit.GetTxBytes")
	}

	txResponse, err := ct.terra.clientCtx.WithBroadcastMode(string(txtypes.BroadcastMode_BROADCAST_MODE_SYNC)).BroadcastTx(txBytes)
	if err != nil {
		return errors.Wrap(err, "error in Transmit.Send")
	}
	ct.terra.Log.Infof("[%s] TX Hash: %s", ct.JobID, txResponse.TxHash)
	return nil
}

// LatestConfigDigestAndEpoch fetches the latest details from contract state
func (ct *Contract) LatestConfigDigestAndEpoch(ctx context.Context) (
	configDigest types.ConfigDigest,
	epoch uint32,
	err error,
) {
	queryParams := NewAbciQueryParams(ct.ContractID.String(), []byte(`"latest_config_digest_and_epoch"`))
	data, err := ct.terra.codec.MarshalJSON(queryParams)
	if err != nil {
		return
	}
	resp, err := ct.terra.clientCtx.QueryABCI(abci.RequestQuery{
		Data:   data,
		Path:   "custom/wasm/contractStore",
		Height: 0,
		Prove:  false,
	})
	if err != nil {
		return types.ConfigDigest{}, 0, err
	}

	var digest LatestConfigDigestAndEpoch
	if err := json.Unmarshal(resp.Value, &digest); err != nil {
		return types.ConfigDigest{}, 0, err
	}

	return digest.ConfigDigest, digest.Epoch, nil
}

func (ct *Contract) FromAccount() types.Account {
	return types.Account(ct.Transmitter.PublicKey().String())
}
