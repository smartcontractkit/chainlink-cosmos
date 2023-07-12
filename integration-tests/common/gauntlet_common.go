package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/smartcontractkit/chainlink-starknet/integration-tests/utils"
)

var (
	ethAddressGoerli = "0x049d36570d4e46f48e99674bd3fcc84644ddd6b96f7c741b1562b82f9e004dc7"
	nAccount         string
)

func (testState *Test) fundNodes() ([]string, error) {
	l := utils.GetTestLogger(testState.T)
	var nAccounts []string
	var err error
	for _, key := range testState.GetNodeKeys() {
		if key.TXKey.Data.Attributes.StarkKey == "" {
			return nil, errors.New("stark key can't be empty")
		}
		nAccount, err = testState.Cg.DeployAccountContract(100, key.TXKey.Data.Attributes.StarkKey)
		if err != nil {
			return nil, err
		}
		nAccounts = append(nAccounts, nAccount)
	}

	if err != nil {
		return nil, err
	}

	for _, key := range nAccounts {
		// We are not deploying in parallel here due to testnet limitations (429 too many requests)
		l.Debug().Msg(fmt.Sprintf("Funding node with address: %s", key))
		_, err = testState.Cg.TransferToken(ethAddressGoerli, key, "100000000000000000") // Transferring 1 ETH to each node
		if err != nil {
			return nil, err
		}
	}

	return nAccounts, nil
}

func (testState *Test) deployLinkToken() error {
	_, err := testState.Cg.UploadContract("cw20_base")
	if err != nil {
		return err
	}

	testState.LinkTokenAddr, err = testState.Cg.DeployLinkTokenContract()
	if err != nil {
		return err
	}
	err = os.Setenv("LINK", testState.LinkTokenAddr)
	if err != nil {
		return err
	}
	return nil
}

func (testState *Test) deployAccessController() error {
	var err error
	testState.AccessControllerAddr, err = testState.Cg.DeployAccessControllerContract()
	if err != nil {
		return err
	}
	err = os.Setenv("BILLING_ACCESS_CONTROLLER", testState.AccessControllerAddr)
	if err != nil {
		return err
	}
	return nil
}

func (testState *Test) setConfigDetails(ocrAddress string) error {
	cfg, err := testState.LoadOCR2Config()
	if err != nil {
		return err
	}
	var parsedConfig []byte
	parsedConfig, err = json.Marshal(cfg)
	if err != nil {
		return err
	}
	_, err = testState.Cg.SetConfigDetails(string(parsedConfig), ocrAddress)
	return err
}

func (testState *Test) DeployGauntlet(minSubmissionValue int64, maxSubmissionValue int64, decimals int, name string, observationPaymentGjuels int64, transmissionPaymentGjuels int64) error {
	err := testState.Cg.InstallDependencies()
	if err != nil {
		return err
	}

	testState.AccountAddresses, err = testState.fundNodes()
	if err != nil {
		return err
	}

	err = testState.deployLinkToken()
	if err != nil {
		return err
	}

	err = testState.deployAccessController()
	if err != nil {
		return err
	}

	testState.OCRAddr, err = testState.Cg.DeployOCR2ControllerContract(minSubmissionValue, maxSubmissionValue, decimals, name, testState.LinkTokenAddr)
	if err != nil {
		return err
	}

	testState.ProxyAddr, err = testState.Cg.DeployOCR2ProxyContract(testState.OCRAddr)
	if err != nil {
		return err
	}
	_, err = testState.Cg.AddAccess(testState.OCRAddr, testState.ProxyAddr)
	if err != nil {
		return err
	}

	_, err = testState.Cg.MintLinkToken(testState.LinkTokenAddr, testState.OCRAddr, "100000000000000000000")
	if err != nil {
		return err
	}
	_, err = testState.Cg.SetOCRBilling(observationPaymentGjuels, transmissionPaymentGjuels, testState.OCRAddr)
	if err != nil {
		return err
	}

	err = testState.setConfigDetails(testState.OCRAddr)
	return err
}
