package gauntlet

import (
	"encoding/json"
	"os"

	"github.com/smartcontractkit/chainlink-testing-framework/gauntlet"
)

var (
	cg *CosmosGauntlet
)

type CosmosGauntlet struct {
	dir     string
	G       *gauntlet.Gauntlet
	gr      *GauntletResponse
	options *gauntlet.ExecCommandOptions
}

// GauntletResponse Default response output for cosmos gauntlet commands
type GauntletResponse struct {
	Responses []struct {
		Tx struct {
			Hash    string `json:"hash"`
			Address string `json:"address"`
			Status  string `json:"status"`
			CodeId  int    `json:"codeId"`

			Tx struct {
				Address         string   `json:"address"`
				Code            string   `json:"code"`
				Result          []string `json:"result"`
				TransactionHash string   `json:"transaction_hash"`
			} `json:"tx"`
		} `json:"tx"`
		Contract string `json:"contract"`
	} `json:"responses"`
}

// NewCosmosGauntlet Creates a default gauntlet config
func NewCosmosGauntlet(workingDir string) (*CosmosGauntlet, error) {
	g, err := gauntlet.NewGauntlet()
	g.SetWorkingDir(workingDir)
	if err != nil {
		return nil, err
	}
	cg = &CosmosGauntlet{
		dir: workingDir,
		G:   g,
		gr:  &GauntletResponse{},
		options: &gauntlet.ExecCommandOptions{
			ErrHandling:       []string{},
			CheckErrorsInRead: true,
		},
	}
	return cg, nil
}

// FetchGauntletJsonOutput Parse gauntlet json response that is generated after yarn gauntlet command execution
func (cg *CosmosGauntlet) FetchGauntletJsonOutput() (*GauntletResponse, error) {
	var payload = &GauntletResponse{}
	gauntletOutput, err := os.ReadFile(cg.dir + "report.json")
	if err != nil {
		return payload, err
	}
	err = json.Unmarshal(gauntletOutput, &payload)
	if err != nil {
		return payload, err
	}
	return payload, nil
}

// SetupNetwork Sets up a new network and sets the NODE_URL for Devnet / Cosmos RPC
func (sg *CosmosGauntlet) SetupNetwork(addr string) error {
	sg.G.AddNetworkConfigVar("NODE_URL", addr)
	err := sg.G.WriteNetworkConfigMap(sg.dir + "packages-ts/gauntlet-cosmos-contracts/networks/")
	if err != nil {
		return err
	}

	return nil
}

func (sg *CosmosGauntlet) InstallDependencies() error {
	sg.G.Command = "yarn"
	_, err := sg.G.ExecCommand([]string{"install"}, *sg.options)
	if err != nil {
		return err
	}
	sg.G.Command = "gauntlet"
	return nil
}

func (sg *CosmosGauntlet) UploadContract(name string) (int, error) {
	_, err := sg.G.ExecCommand([]string{"upload", name}, *sg.options)
	if err != nil {
		return 0, err
	}
	sg.gr, err = sg.FetchGauntletJsonOutput()
	if err != nil {
		return 0, err
	}
	return sg.gr.Responses[0].Tx.CodeId, nil
}

// func (sg *CosmosGauntlet) DeployAccountContract(salt int64, pubKey string) (string, error) {
// 	_, err := sg.G.ExecCommand([]string{"account:deploy", fmt.Sprintf("--salt=%d", salt), fmt.Sprintf("--publicKey=%s", pubKey)}, *sg.options)
// 	if err != nil {
// 		return "", err
// 	}
// 	sg.gr, err = sg.FetchGauntletJsonOutput()
// 	if err != nil {
// 		return "", err
// 	}
// 	return sg.gr.Responses[0].Contract, nil
// }

func (sg *CosmosGauntlet) DeployLinkTokenContract() (string, error) {
	_, err := sg.G.ExecCommand([]string{"token:deploy"}, *sg.options)
	if err != nil {
		return "", err
	}
	sg.gr, err = sg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return sg.gr.Responses[0].Contract, nil
}

// func (sg *CosmosGauntlet) MintLinkToken(token, to, amount string) (string, error) {
// 	_, err := sg.G.ExecCommand([]string{"ERC20:mint", fmt.Sprintf("--account=%s", to), fmt.Sprintf("--amount=%s", amount), token}, *sg.options)
// 	if err != nil {
// 		return "", err
// 	}
// 	sg.gr, err = sg.FetchGauntletJsonOutput()
// 	if err != nil {
// 		return "", err
// 	}
// 	return sg.gr.Responses[0].Contract, nil
// }

func (sg *CosmosGauntlet) TransferToken(token, to, amount string) (string, error) {
	_, err := sg.G.ExecCommand([]string{"ERC20:transfer", fmt.Sprintf("--recipient=%s", to), fmt.Sprintf("--amount=%s", amount), token}, *sg.options)
	if err != nil {
		return "", err
	}
	sg.gr, err = sg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return sg.gr.Responses[0].Contract, nil
}

// func (sg *CosmosGauntlet) DeployOCR2ControllerContract(minSubmissionValue int64, maxSubmissionValue int64, decimals int, name string, linkTokenAddress string) (string, error) {
// 	_, err := sg.G.ExecCommand([]string{"ocr2:deploy", fmt.Sprintf("--minSubmissionValue=%d", minSubmissionValue), fmt.Sprintf("--maxSubmissionValue=%d", maxSubmissionValue), fmt.Sprintf("--decimals=%d", decimals), fmt.Sprintf("--name=%s", name), fmt.Sprintf("--link=%s", linkTokenAddress)}, *sg.options)
// 	if err != nil {
// 		return "", err
// 	}
// 	sg.gr, err = sg.FetchGauntletJsonOutput()
// 	if err != nil {
// 		return "", err
// 	}
// 	return sg.gr.Responses[0].Contract, nil
// }

// func (sg *CosmosGauntlet) DeployAccessControllerContract() (string, error) {
// 	_, err := sg.G.ExecCommand([]string{"access_controller:deploy"}, *sg.options)
// 	if err != nil {
// 		return "", err
// 	}
// 	sg.gr, err = sg.FetchGauntletJsonOutput()
// 	if err != nil {
// 		return "", err
// 	}
// 	return sg.gr.Responses[0].Contract, nil
// }

// func (sg *CosmosGauntlet) DeployOCR2ProxyContract(aggregator string) (string, error) {
// 	_, err := sg.G.ExecCommand([]string{"proxy:deploy", fmt.Sprintf("--address=%s", aggregator)}, *sg.options)
// 	if err != nil {
// 		return "", err
// 	}
// 	sg.gr, err = sg.FetchGauntletJsonOutput()
// 	if err != nil {
// 		return "", err
// 	}
// 	return sg.gr.Responses[0].Contract, nil
// }

// func (sg *CosmosGauntlet) SetOCRBilling(observationPaymentGjuels int64, transmissionPaymentGjuels int64, ocrAddress string) (string, error) {
// 	_, err := sg.G.ExecCommand([]string{"ocr2:set_billing", fmt.Sprintf("--observationPaymentGjuels=%d", observationPaymentGjuels), fmt.Sprintf("--transmissionPaymentGjuels=%d", transmissionPaymentGjuels), ocrAddress}, *sg.options)
// 	if err != nil {
// 		return "", err
// 	}
// 	sg.gr, err = sg.FetchGauntletJsonOutput()
// 	if err != nil {
// 		return "", err
// 	}
// 	return sg.gr.Responses[0].Contract, nil
// }

// func (sg *CosmosGauntlet) SetConfigDetails(cfg string, ocrAddress string) (string, error) {
// 	_, err := sg.G.ExecCommand([]string{"ocr2:set_config", "--input=" + cfg, ocrAddress}, *sg.options)
// 	if err != nil {
// 		return "", err
// 	}
// 	sg.gr, err = sg.FetchGauntletJsonOutput()
// 	if err != nil {
// 		return "", err
// 	}
// 	return sg.gr.Responses[0].Contract, nil
// }

// func (sg *CosmosGauntlet) AddAccess(aggregator, address string) (string, error) {
// 	_, err := sg.G.ExecCommand([]string{"ocr2:add_access", fmt.Sprintf("--address=%s", address), aggregator}, *sg.options)
// 	if err != nil {
// 		return "", err
// 	}
// 	sg.gr, err = sg.FetchGauntletJsonOutput()
// 	if err != nil {
// 		return "", err
// 	}
// 	return sg.gr.Responses[0].Contract, nil
// }
