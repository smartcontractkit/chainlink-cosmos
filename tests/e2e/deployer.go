package e2e

import (
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/actypes"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/cw20types"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/flagstypes"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/ocr2proxytypes"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/ocr2types"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/utils"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/validatortypes"
	"github.com/smartcontractkit/integrations-framework/client"
	"github.com/smartcontractkit/terra.go/msg"
	"path/filepath"
)

// ContractDeployer provides the implementations for deploying Terra based contracts
type ContractDeployer struct {
	client *TerraLCDClient
}

func (t *ContractDeployer) DeployOCRv2Validator(threshold uint32, flags string) (*OCRv2Validator, error) {
	contractAddr, err := t.client.Instantiate(
		filepath.Join(utils.ContractsDir, "deviation_flagging_validator.wasm"),
		validatortypes.InstantiateMsg{
			FlaggingThreshold: threshold,
			Flags:             flags,
		},
	)
	if err != nil {
		return nil, err
	}
	ca, err := msg.AccAddressFromBech32(contractAddr)
	if err != nil {
		return nil, err
	}
	return &OCRv2Validator{
		client:  t.client,
		address: ca,
	}, nil
}

func (t *ContractDeployer) DeployOCRv2Proxy(addr string) (*OCRv2Proxy, error) {
	proxyAddr, err := t.client.Instantiate(
		filepath.Join(utils.ContractsDir, "proxy_ocr2.wasm"),
		ocr2proxytypes.InstantiateMsg{ContractAddress: addr},
	)
	if err != nil {
		return nil, err
	}
	proxAddr, err := msg.AccAddressFromBech32(proxyAddr)
	if err != nil {
		return nil, err
	}
	return &OCRv2Proxy{
		client:  t.client,
		address: proxAddr,
	}, nil
}

func (t *ContractDeployer) DeployOCRv2Flags(lowAC string, raiseAC string) (*OCRv2Flags, error) {
	contractAddr, err := t.client.Instantiate(
		filepath.Join(utils.ContractsDir, "flags.wasm"),
		flagstypes.InstantiateMsg{
			LoweringAccessController: lowAC,
			RaisingAccessController:  raiseAC,
		},
	)
	if err != nil {
		return nil, err
	}
	ca, err := msg.AccAddressFromBech32(contractAddr)
	if err != nil {
		return nil, err
	}
	return &OCRv2Flags{
		client:  t.client,
		address: ca,
	}, nil
}

func (t *ContractDeployer) DeployOCRv2ValidatorProxy(addr string) (*OCRv2Proxy, error) {
	proxyAddr, err := t.client.Instantiate(
		filepath.Join(utils.ContractsDir, "proxy_validator.wasm"),
		ocr2proxytypes.InstantiateMsg{ContractAddress: addr},
	)
	if err != nil {
		return nil, err
	}
	proxAddr, err := msg.AccAddressFromBech32(proxyAddr)
	if err != nil {
		return nil, err
	}
	return &OCRv2Proxy{
		client:  t.client,
		address: proxAddr,
	}, nil
}

func NewTerraContractDeployer(client client.BlockchainClient) *ContractDeployer {
	return &ContractDeployer{
		client.(*TerraLCDClient),
	}
}

func (t *ContractDeployer) DeployLinkTokenContract() (*LinkToken, error) {
	linkAddr, err := t.client.Instantiate(
		filepath.Join(utils.CommonContractsDir, "cw20_base.wasm"),
		cw20types.InstantiateMsg{
			Name:     "LinkToken",
			Symbol:   "LINK",
			Decimals: 18,
			InitialBalances: []cw20types.InitialBalanceMsg{
				{
					Address: t.client.DefaultWallet.AccAddress,
					Amount:  "9000000000000000000",
				},
			},
		})
	if err != nil {
		return nil, err
	}
	addr, err := msg.AccAddressFromBech32(linkAddr)
	if err != nil {
		return nil, err
	}
	return &LinkToken{
		client:  t.client,
		address: addr,
	}, nil
}

func (t *ContractDeployer) DeployOCRv2(paymentControllerAddr string, requesterControllerAddr string, linkTokenAddr string) (*OCRv2, error) {
	ocr2, err := t.client.Instantiate(
		filepath.Join(utils.ContractsDir, "ocr2.wasm"),
		ocr2types.OCRv2InstantiateMsg{
			BillingAccessController:   paymentControllerAddr,
			RequesterAccessController: requesterControllerAddr,
			LinkToken:                 linkTokenAddr,
			Decimals:                  8,
			Description:               "ETH/USD",
			MinAnswer:                 "1",
			MaxAnswer:                 "10",
		})
	if err != nil {
		return nil, err
	}
	addr, err := msg.AccAddressFromBech32(ocr2)
	if err != nil {
		return nil, err
	}
	return &OCRv2{
		client:  t.client,
		address: addr,
	}, nil
}

func (t *ContractDeployer) DeployOCRv2AccessController() (*AccessController, error) {
	acAddr, err := t.client.Instantiate(
		filepath.Join(utils.ContractsDir, "access_controller.wasm"),
		actypes.InstantiateMsg{},
	)
	if err != nil {
		return nil, err
	}
	addr, err := msg.AccAddressFromBech32(acAddr)
	if err != nil {
		return nil, err
	}
	return &AccessController{
		client:  t.client,
		address: addr,
	}, nil
}
