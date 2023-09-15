package cosmwasm

import (
	"context"
	"encoding/binary"
	"encoding/json"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	cosmosSDK "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/chainlink-relay/pkg/logger"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/config"

	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

// A Uint128 is an unsigned 128-bit number.
type Uint128 struct {
	Lo, Hi uint64
}

// Big endian order
func (u Uint128) Bytes(b []byte) {
	binary.BigEndian.PutUint64(b[:8], u.Hi)
	binary.BigEndian.PutUint64(b[8:], u.Lo)
}

var _ types.ContractTransmitter = (*ContractTransmitter)(nil)

type ContractTransmitter struct {
	*OCR2Reader
	msgEnqueuer adapters.MsgEnqueuer
	lggr        logger.Logger
	jobID       string
	contract    cosmosSDK.AccAddress
	sender      cosmosSDK.AccAddress
	cfg         config.Config
}

func NewContractTransmitter(
	reader *OCR2Reader,
	jobID string,
	contract cosmosSDK.AccAddress,
	sender cosmosSDK.AccAddress,
	msgEnqueuer adapters.MsgEnqueuer,
	lggr logger.Logger,
	cfg config.Config,
) *ContractTransmitter {
	return &ContractTransmitter{
		OCR2Reader:  reader,
		jobID:       jobID,
		contract:    contract,
		msgEnqueuer: msgEnqueuer,
		sender:      sender,
		lggr:        lggr,
		cfg:         cfg,
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
	// TODO: This temporarily appends a dummy gas price to the report.
	// When core/relayer is updated to support adding gas price to the report, we can remove this
	// dummyGasPrice_u128 := Uint128{Lo: 1, Hi: 0}
	// dummyGasPrice := make([]byte, 16)
	// dummyGasPrice_u128.Bytes(dummyGasPrice)
	// msgStruct.Transmit.Report = append([]byte(report), dummyGasPrice...)

	dummyReport := []byte{97, 91, 43, 83}                                                                                                        // observationTimestamp
	dummyReport = append(dummyReport, []byte{0, 1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}...) // observers
	dummyReport = append(dummyReport, 2)                                                                                                         // observationLen
	dummyReport = append(dummyReport, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 150, 2, 210}...)                                            // observation1
	dummyReport = append(dummyReport, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 150, 2, 210}...)                                            // observation2
	dummyReport = append(dummyReport, []byte{0, 0, 0, 0, 0, 0, 0, 0, 13, 224, 182, 179, 167, 100, 0, 0}...)                                      // juelsPerFeeCoin
	dummyReport = append(dummyReport, []byte{0, 0, 0, 0, 0, 0, 0, 0, 13, 224, 182, 179, 167, 100, 0, 0}...)                                      // gasPrice
	msgStruct.Transmit.Report = dummyReport

	for _, sig := range sigs {
		msgStruct.Transmit.Signatures = append(msgStruct.Transmit.Signatures, sig.Signature)
	}
	msgBytes, err := json.Marshal(msgStruct)
	if err != nil {
		return err
	}
	m := &wasmtypes.MsgExecuteContract{
		Sender:   ct.sender.String(),
		Contract: ct.contract.String(),
		Msg:      msgBytes,
		Funds:    cosmosSDK.Coins{},
	}
	_, err = ct.msgEnqueuer.Enqueue(ct.contract.String(), m)
	return err
}

func (ct *ContractTransmitter) FromAccount() (types.Account, error) {
	return types.Account(ct.sender.String()), nil
}
