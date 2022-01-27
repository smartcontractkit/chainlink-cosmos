package e2e

import (
	"context"
	"encoding/json"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/ocr2types"
	"github.com/smartcontractkit/integrations-framework/contracts"
	"github.com/smartcontractkit/libocr/offchainreporting2/confighelper"
	terraClient "github.com/smartcontractkit/terra.go/client"
	"github.com/smartcontractkit/terra.go/msg"
	"strconv"
)

// OCRv2 represents a OVR v2 contract deployed on terra as WASM
type OCRv2 struct {
	client  *TerraLCDClient
	address msg.AccAddress
}

func (t *OCRv2) ProgramAddress() string {
	panic("implement me")
}

func (t *OCRv2) TransmissionsAddr() string {
	panic("implement me")
}

func (t *OCRv2) DumpState() error {
	panic("implement me")
}

func (t *OCRv2) GetContractData(ctx context.Context) (*contracts.OffchainAggregatorData, error) {
	panic("implement me")
}

func (t *OCRv2) AuthorityAddr(s string) (string, error) {
	panic("implement me")
}

func (t *OCRv2) SetValidatorConfig(flaggingThreshold uint32, validatorAddr string) error {
	panic("implement me")
}

func (t *OCRv2) SetBilling(op uint32, tp uint32, controllerAddr string) error {
	sender := t.client.DefaultWallet.AccAddress
	executeMsg := ocr2types.ExecuteSetBillingMsg{
		SetBilling: ocr2types.ExecuteSetBillingMsgType{
			Config: ocr2types.ExecuteSetBillingConfigMsgType{
				ObservationPayment:  op,
				RecommendedGasPrice: 1,
			},
		}}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	_, err = t.client.SendTX(terraClient.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgExecuteContract(
				sender,
				t.address,
				executeMsgBytes,
				msg.NewCoins(),
			),
		},
	}, true)
	if err != nil {
		return err
	}
	return nil
}

func (t *OCRv2) GetLatestRoundData() (uint64, uint64, uint64, error) {
	resp := ocr2types.QueryLatestRoundDataResponse{}
	if err := t.client.QuerySmart(context.Background(), t.address, ocr2types.QueryLatestRoundData, &resp); err != nil {
		return 0, 0, 0, err
	}
	answer, _ := strconv.Atoi(resp.QueryResult.Answer)
	return uint64(answer), resp.QueryResult.TransmissionTimestamp, resp.QueryResult.RoundID, nil
}

func (t *OCRv2) SetOracles(ocParams contracts.OffChainAggregatorV2Config) error {
	panic("implement me")
}

func (t *OCRv2) SetOffChainConfig(cfg contracts.OffChainAggregatorV2Config) error {
	sender := t.client.DefaultWallet.AccAddress
	signers, transmitters, f, onChainCfg, version, offChainConfigBytes, err := confighelper.ContractSetConfigArgsForTests(
		cfg.DeltaProgress,
		cfg.DeltaResend,
		cfg.DeltaRound,
		cfg.DeltaGrace,
		cfg.DeltaStage,
		cfg.RMax,
		cfg.S,
		cfg.Oracles,
		cfg.ReportingPluginConfig,
		cfg.MaxDurationQuery,
		cfg.MaxDurationObservation,
		cfg.MaxDurationReport,
		cfg.MaxDurationShouldAcceptFinalizedReport,
		cfg.MaxDurationShouldTransmitAcceptedReport,
		cfg.F,
		cfg.OnchainConfig,
	)
	if err != nil {
		return err
	}
	// convert type for marshalling
	signerArray := [][]byte{}
	transmitterArray := []string{}
	for i := 0; i < len(signers); i++ {
		signerArray = append(signerArray, signers[i])
		transmitterArray = append(transmitterArray, string(transmitters[i]))
	}

	tx := ocr2types.ExecuteSetConfig{
		SetConfig: ocr2types.SetConfigDetails{
			Signers:               signerArray,
			Transmitters:          transmitterArray,
			F:                     f,
			OnchainConfig:         onChainCfg,
			OffchainConfigVersion: version,
			OffchainConfig:        offChainConfigBytes,
		},
	}
	executeMsgBytes, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	_, err = t.client.SendTX(terraClient.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgExecuteContract(
				sender,
				t.address,
				executeMsgBytes,
				msg.NewCoins(),
			),
		},
	}, true)
	if err != nil {
		return err
	}
	return nil
}

func (t *OCRv2) RequestNewRound() error {
	panic("implement me")
}

func (t *OCRv2) GetOwedPayment(transmitter string) (map[string]interface{}, error) {
	transmitterAddr, _ := msg.AccAddressFromBech32(transmitter)
	resp := make(map[string]interface{})
	if err := t.client.QuerySmart(
		context.Background(),
		t.address,
		ocr2types.QueryOwedPaymentMsg{
			OwedPayment: ocr2types.QueryOwedPaymentTypeMsg{
				Transmitter: transmitterAddr,
			},
		},
		&resp,
	); err != nil {
		return nil, err
	}
	return resp, nil
}

func (t *OCRv2) GetRoundData(roundID uint32) (map[string]interface{}, error) {
	resp := make(map[string]interface{})
	if err := t.client.QuerySmart(
		context.Background(),
		t.address,
		ocr2types.QueryRoundDataMsg{
			RoundData: ocr2types.QueryRoundDataTypeMsg{
				RoundID: roundID,
			},
		},
		&resp,
	); err != nil {
		return nil, err
	}
	return resp, nil
}

func (t *OCRv2) GetLatestConfigDetails() (map[string]interface{}, error) {
	resp := make(map[string]interface{})
	if err := t.client.QuerySmart(context.Background(), t.address, ocr2types.QueryLatestConfigDetails, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (t *OCRv2) TransferOwnership(to string) error {
	sender := t.client.DefaultWallet.AccAddress
	toAddr, _ := msg.AccAddressFromHex(to)
	executeMsg := ocr2types.ExecuteTransferOwnershipMsg{
		TransferOwnership: ocr2types.ExecuteTransferOwnershipMsgType{
			To: toAddr,
		}}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	_, err = t.client.SendTX(terraClient.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgExecuteContract(
				sender,
				t.address,
				executeMsgBytes,
				msg.NewCoins(),
			),
		},
	}, true)
	if err != nil {
		return err
	}
	return nil
}

func (t *OCRv2) Address() string {
	return t.address.String()
}
