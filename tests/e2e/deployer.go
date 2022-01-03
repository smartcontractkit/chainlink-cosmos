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
	"github.com/smartcontractkit/integrations-framework/contracts"
	"github.com/smartcontractkit/terra.go/msg"
	"math/big"
	"path/filepath"
)

// ContractDeployer provides the implementations for deploying Terra based contracts
type ContractDeployer struct {
	client *TerraLCDClient
}

func (t *ContractDeployer) DeployOCRv2Validator(threshold uint32, flags string) (contracts.OCRv2Flags, error) {
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

func (t *ContractDeployer) DeployOCRv2Proxy(addr string) (contracts.OCRv2Proxy, error) {
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

func (t *ContractDeployer) DeployOCRv2Flags(lowAC string, raiseAC string) (contracts.OCRv2Flags, error) {
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

func (t *ContractDeployer) DeployOCRv2ValidatorProxy(addr string) (contracts.OCRv2Proxy, error) {
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

func (t *ContractDeployer) DeployOCRv2Store(billingAC string) (contracts.OCRv2Store, error) {
	panic("implement me")
}

func NewTerraContractDeployer(client client.BlockchainClient) *ContractDeployer {
	return &ContractDeployer{
		client.(*TerraLCDClient),
	}
}

func (t *ContractDeployer) DeployLinkTokenContract() (contracts.LinkToken, error) {
	linkAddr, err := t.client.Instantiate(
		filepath.Join(utils.CommonContractsDir, "cw20_base.wasm"),
		cw20types.InstantiateMsg{
			Name:     "LinkToken",
			Symbol:   "LINK",
			Decimals: 18,
			InitialBalances: []cw20types.InitialBalanceMsg{
				{
					Address: t.client.DefaultWallet.AccAddress,
					Amount:  "1000000000",
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

func (t *ContractDeployer) DeployOCRv2(paymentControllerAddr string, requesterControllerAddr string, linkTokenAddr string) (contracts.OCRv2, error) {
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

func (t *ContractDeployer) DeployOCRv2AccessController() (contracts.OCRv2AccessController, error) {
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

func (t *ContractDeployer) DeployOffChainAggregator(linkAddr string, offchainOptions contracts.OffchainOptions) (contracts.OffchainAggregator, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployVRFContract() (contracts.VRF, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployMockETHLINKFeed(answer *big.Int) (contracts.MockETHLINKFeed, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployMockGasFeed(answer *big.Int) (contracts.MockGasFeed, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployUpkeepRegistrationRequests(linkAddr string, minLinkJuels *big.Int) (contracts.UpkeepRegistrar, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployKeeperRegistry(opts *contracts.KeeperRegistryOpts) (contracts.KeeperRegistry, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployKeeperConsumer(updateInterval *big.Int) (contracts.KeeperConsumer, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployVRFConsumer(linkAddr string, coordinatorAddr string) (contracts.VRFConsumer, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployVRFCoordinator(linkAddr string, bhsAddr string) (contracts.VRFCoordinator, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployBlockhashStore() (contracts.BlockHashStore, error) {
	panic("implement me")
}

func (t *ContractDeployer) Balance() (*big.Float, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployStorageContract() (contracts.Storage, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployAPIConsumer(linkAddr string) (contracts.APIConsumer, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployOracle(linkAddr string) (contracts.Oracle, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployReadAccessController() (contracts.ReadAccessController, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployFlags(rac string) (contracts.Flags, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployDeviationFlaggingValidator(flags string, flaggingThreshold *big.Int) (contracts.DeviationFlaggingValidator, error) {
	panic("implement me")
}

func (t *ContractDeployer) DeployFluxAggregatorContract(linkAddr string, fluxOptions contracts.FluxAggregatorOptions) (contracts.FluxAggregator, error) {
	panic("implement me")
}
