package e2e

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/ocr2proxytypes"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/ocr2types"
	terraClient "github.com/smartcontractkit/terra.go/client"
	"github.com/smartcontractkit/terra.go/msg"
)

type OCRv2Proxy struct {
	client  *TerraLCDClient
	address msg.AccAddress
}

func (m *OCRv2Proxy) Address() string {
	return m.address.String()
}

func (m *OCRv2Proxy) ProposeContract(addr string) error {
	executeMsg := ocr2proxytypes.ProposeContractMsg{
		ContractAddress: addr,
	}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	return m.send(executeMsgBytes)
}

func (m *OCRv2Proxy) ConfirmContract(addr string) error {
	executeMsg := ocr2proxytypes.ConfirmContractMsg{
		ContractAddress: addr,
	}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	return m.send(executeMsgBytes)
}

func (m *OCRv2Proxy) TransferOwnership(to string) error {
	executeMsg := ocr2proxytypes.TransferOwnershipMsg{
		ToAddress: to,
	}
	executeMsgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return err
	}
	return m.send(executeMsgBytes)
}

func (m *OCRv2Proxy) send(executeMsgBytes []byte) error {
	sender := m.client.DefaultWallet.AccAddress
	_, err := m.client.SendTX(terraClient.CreateTxOptions{
		Msgs: []msg.Msg{
			msg.NewMsgExecuteContract(
				sender,
				m.address,
				executeMsgBytes,
				msg.NewCoins(),
			),
		},
	}, true)
	return err
}

func (m *OCRv2Proxy) GetLatestRoundData() (uint64, uint64, uint64, error) {
	resp := ocr2types.QueryLatestRoundDataResponse{}
	log.Warn().Interface("Addr", m.address)
	if err := m.client.QuerySmart(context.Background(), m.address, ocr2types.QueryLatestRoundData, &resp); err != nil {
		return 0, 0, 0, err
	}
	answer, _ := strconv.Atoi(resp.QueryResult.Answer)
	return uint64(answer), resp.QueryResult.TransmissionTimestamp, resp.QueryResult.RoundID, nil
}

func (m *OCRv2Proxy) GetRoundData(roundID uint32) (map[string]interface{}, error) {
	resp := make(map[string]interface{})
	if err := m.client.QuerySmart(
		context.Background(),
		m.address,
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

func (m *OCRv2Proxy) GetDecimals() (int, error) {
	resp := make(map[string]int)
	if err := m.client.QuerySmart(
		context.Background(),
		m.address,
		ocr2types.QueryDecimals,
		&resp,
	); err != nil {
		return 0, err
	}
	log.Info().Interface("Description response", resp).Msg("The decimals from the proxy")
	return resp["query_result"], nil
}

func (m *OCRv2Proxy) GetDescription() (string, error) {
	resp := make(map[string]string)
	if err := m.client.QuerySmart(
		context.Background(),
		m.address,
		ocr2types.QueryDescription,
		&resp,
	); err != nil {
		return "", err
	}
	log.Info().Interface("Description response", resp).Msg("The description from the proxy")
	return resp["query_result"], nil
}
