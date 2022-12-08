package e2e

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strconv"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/rs/zerolog/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/ocr2types"
	"github.com/smartcontractkit/chainlink/integration-tests/contracts"
	"github.com/smartcontractkit/libocr/offchainreporting2/confighelper"
	terraClient "github.com/smartcontractkit/terra.go/client"
)

// OCRv2 represents a OVR v2 contract deployed on terra as WASM
type OCRv2 struct {
	Client *TerraLCDClient
	Addr   sdk.AccAddress
}

// SetValidatorConfig sets validator config
func (t *OCRv2) SetValidatorConfig(gasLimit uint64, validatorAddr string) error {
	sender := t.Client.DefaultWallet.AccAddress
	executeMsg := ocr2types.ExecuteSetValidator{
		SetValidator: ocr2types.ExecuteSetValidatorConfig{
			Config: ocr2types.ExecuteSetValidatorConfigType{
				Address:  validatorAddr,
				GasLimit: gasLimit,
			},
		}}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	_, err = t.Client.SendTX(terraClient.CreateTxOptions{
		Msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{
				Sender:   sender.String(),
				Contract: t.Addr.String(),
				Msg:      executeMsgBytes,
				Funds:    sdk.NewCoins(),
			},
		},
	}, true)
	if err != nil {
		return err
	}
	return nil
}

// SetBilling sets billing params for OCR
func (t *OCRv2) SetBilling(baseGas uint64, op uint64, tp uint64, recommendedGasPriceULuna string, controllerAddr string) error {
	sender := t.Client.DefaultWallet.AccAddress
	executeMsg := ocr2types.ExecuteSetBillingMsg{
		SetBilling: ocr2types.ExecuteSetBillingMsgType{
			Config: ocr2types.ExecuteSetBillingConfigMsgType{
				BaseGas:             baseGas,
				TransmissionPayment: tp,
				ObservationPayment:  op,
				RecommendedGasPrice: recommendedGasPriceULuna,
			},
		}}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	_, err = t.Client.SendTX(terraClient.CreateTxOptions{
		Msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{
				Sender:   sender.String(),
				Contract: t.Addr.String(),
				Msg:      executeMsgBytes,
				Funds:    sdk.NewCoins(),
			},
		},
	}, true)
	if err != nil {
		return err
	}
	return nil
}

func (t *OCRv2) GetLatestRoundData() (uint64, uint64, uint64, error) {
	resp := ocr2types.QueryLatestRoundDataResponse{}
	log.Warn().Interface("Addr", t.Addr)
	if err := t.Client.QuerySmart(context.Background(), t.Addr, ocr2types.QueryLatestRoundData, &resp); err != nil {
		return 0, 0, 0, err
	}
	answer, _ := strconv.Atoi(resp.QueryResult.Answer)
	return uint64(answer), resp.QueryResult.TransmissionTimestamp, resp.QueryResult.RoundID, nil
}

// SetOffChainConfig sets offchain config
func (t *OCRv2) SetOffChainConfig(cfg contracts.OffChainAggregatorV2Config) ([]string, error) {
	sender := t.Client.DefaultWallet.AccAddress
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
		return nil, err
	}
	// convert type for marshalling
	signerArray := [][]byte{}
	transmitterArray := []string{}
	for i := 0; i < len(signers); i++ {
		signerArray = append(signerArray, signers[i])
		transmitterArray = append(transmitterArray, string(transmitters[i]))
	}

	msgBytes, err := json.Marshal(ocr2types.BeginProposal)
	if err != nil {
		return nil, err
	}
	txResp, err := t.Client.SendTX(terraClient.CreateTxOptions{
		Msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{
				Sender:   sender.String(),
				Contract: t.Addr.String(),
				Msg:      msgBytes,
				Funds:    sdk.NewCoins(),
			},
		},
	}, true)
	if err != nil {
		return nil, err
	}
	proposalID, err := t.Client.GetEventAttrValue(txResp, "proposal_id")
	if err != nil {
		return nil, err
	}
	payees := make([]string, 0)
	for i := 0; i < len(transmitterArray); i++ {
		payees = append(payees, sender.String())
	}
	proposeConfigBytes, err := json.Marshal(
		ocr2types.ProposeConfig{
			ProposeConfig: ocr2types.ProposeConfigDetails{
				ID:            proposalID,
				Payees:        payees,
				Signers:       signerArray,
				Transmitters:  transmitterArray,
				F:             f,
				OnchainConfig: onChainCfg,
			}})
	if err != nil {
		return nil, err
	}
	_, err = t.Client.SendTX(terraClient.CreateTxOptions{
		Msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{
				Sender:   sender.String(),
				Contract: t.Addr.String(),
				Msg:      proposeConfigBytes,
				Funds:    sdk.NewCoins(),
			},
		},
	}, true)
	if err != nil {
		return nil, err
	}
	proposeOffchainCfgBytes, err := json.Marshal(
		ocr2types.ProposeOffchainConfig{
			ProposeOffchainConfig: ocr2types.ProposeOffchainConfigDetails{
				ID:                    proposalID,
				OffchainConfigVersion: version,
				OffchainConfig:        offChainConfigBytes,
			}})
	if err != nil {
		return nil, err
	}
	_, err = t.Client.SendTX(terraClient.CreateTxOptions{
		Msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{
				Sender:   sender.String(),
				Contract: t.Addr.String(),
				Msg:      proposeOffchainCfgBytes,
				Funds:    sdk.NewCoins(),
			},
		},
	}, true)
	if err != nil {
		return nil, err
	}
	finalizeBytes, err := json.Marshal(
		ocr2types.FinalizeProposal{
			FinalizeProposal: ocr2types.FinalizeProposalDetails{
				ID: proposalID,
			}})
	if err != nil {
		return nil, err
	}
	respWithDigest, err := t.Client.SendTX(terraClient.CreateTxOptions{
		Msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{
				Sender:   sender.String(),
				Contract: t.Addr.String(),
				Msg:      finalizeBytes,
				Funds:    sdk.NewCoins(),
			},
		},
	}, true)
	if err != nil {
		return nil, err
	}
	proposalDigest, err := t.Client.GetEventAttrValue(respWithDigest, "digest")
	if err != nil {
		return nil, err
	}
	digHex, err := hex.DecodeString(proposalDigest)
	if err != nil {
		return nil, err
	}
	acceptProposalBytes, err := json.Marshal(
		ocr2types.AcceptProposal{
			AcceptProposal: ocr2types.AcceptProposalDetails{
				ID:     proposalID,
				Digest: digHex,
			}})
	if err != nil {
		return nil, err
	}
	_, err = t.Client.SendTX(terraClient.CreateTxOptions{
		Msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{
				Sender:   sender.String(),
				Contract: t.Addr.String(),
				Msg:      acceptProposalBytes,
				Funds:    sdk.NewCoins(),
			},
		},
	}, true)
	if err != nil {
		return nil, err
	}
	return transmitterArray, nil
}

func (t *OCRv2) RequestNewRound() error {
	panic("implement me")
}

func (t *OCRv2) GetOwedPayment(transmitter string) (map[string]interface{}, error) {
	transmitterAddr, _ := sdk.AccAddressFromBech32(transmitter)
	resp := make(map[string]interface{})
	if err := t.Client.QuerySmart(
		context.Background(),
		t.Addr,
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
	if err := t.Client.QuerySmart(
		context.Background(),
		t.Addr,
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
	if err := t.Client.QuerySmart(context.Background(), t.Addr, ocr2types.QueryLatestConfigDetails, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (t *OCRv2) TransferOwnership(to string) error {
	sender := t.Client.DefaultWallet.AccAddress
	toAddr, _ := sdk.AccAddressFromHex(to)
	executeMsg := ocr2types.ExecuteTransferOwnershipMsg{
		TransferOwnership: ocr2types.ExecuteTransferOwnershipMsgType{
			To: toAddr,
		}}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	_, err = t.Client.SendTX(terraClient.CreateTxOptions{
		Msgs: []sdk.Msg{
			&wasmtypes.MsgExecuteContract{
				Sender:   sender.String(),
				Contract: t.Addr.String(),
				Msg:      executeMsgBytes,
				Funds:    sdk.NewCoins(),
			},
		},
	}, true)
	if err != nil {
		return err
	}
	return nil
}

// Address gets OCR2 Address
func (t *OCRv2) Address() string {
	return t.Addr.String()
}
