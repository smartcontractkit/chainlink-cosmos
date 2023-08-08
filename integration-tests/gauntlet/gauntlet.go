package gauntlet

import (
	"encoding/json"
	"fmt"
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
			Logs            interface{} `json:"logs"`
			Height          int         `json:"height"`
			TransactionHash string      `json:"transactionHash"`
			Events          []struct {
				Type       string `json:"type"`
				Attributes []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"attributes"`
			} `json:"events"`
			GasWanted int `json:"gasWanted"`
			GasUsed   int `json:"gasUsed"`
			CodeId    int `json:"codeId"` // only present in upload commands
		} `json:"tx"`
		Contract string `json:"contract"`
	} `json:"responses"`
	Data map[string]interface{} `json:"data"`
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
func (cg *CosmosGauntlet) SetupNetwork(nodeUrl string, mnemonic string) error {
	cg.G.AddNetworkConfigVar("NODE_URL", nodeUrl)
	cg.G.AddNetworkConfigVar("MNEMONIC", mnemonic)
	err := cg.G.WriteNetworkConfigMap(cg.dir + "packages-ts/gauntlet-cosmos-contracts/networks/")
	if err != nil {
		return err
	}

	return nil
}

func (cg *CosmosGauntlet) InstallDependencies() error {
	cg.G.Command = "yarn"
	_, err := cg.G.ExecCommand([]string{"install"}, *cg.options)
	if err != nil {
		return err
	}
	cg.G.Command = "gauntlet"
	return nil
}

func (cg *CosmosGauntlet) UploadContracts(names []string) (int, error) {
	if names == nil {
		names = []string{}
	}
	_, err := cg.G.ExecCommand(append([]string{"upload"}, names...), *cg.options)
	if err != nil {
		return 0, err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return 0, err
	}
	return cg.gr.Responses[0].Tx.CodeId, nil
}

//func (cg *CosmosGauntlet) DeployAccountContract(salt int64, pubKey string) (string, error) {
//_, err := cg.G.ExecCommand([]string{"account:deploy", fmt.Sprintf("--salt=%d", salt), fmt.Sprintf("--publicKey=%s", pubKey)}, *cg.options)
//if err != nil {
//return "", err
//}
//cg.gr, err = cg.FetchGauntletJsonOutput()
//if err != nil {
//return "", err
//}
//return cg.gr.Responses[0].Contract, nil
//}

func (cg *CosmosGauntlet) DeployLinkTokenContract() (string, error) {
	_, err := cg.G.ExecCommand([]string{"token:deploy"}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Responses[0].Contract, nil
}

func (cg *CosmosGauntlet) MintLinkToken(token, to, amount string) (string, error) {
	_, err := cg.G.ExecCommand([]string{"cw20_base:mint", fmt.Sprintf("--to=%s", to), fmt.Sprintf("--amount=%s", amount), token}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Responses[0].Contract, nil
}

func (cg *CosmosGauntlet) TransferToken(token, to, amount string) (string, error) {
	_, err := cg.G.ExecCommand([]string{"ERC20:transfer", fmt.Sprintf("--recipient=%s", to), fmt.Sprintf("--amount=%s", amount), token}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Responses[0].Contract, nil
}

func (cg *CosmosGauntlet) DeployOCR2ControllerContract(minSubmissionValue int64, maxSubmissionValue int64, decimals int, name string, linkTokenAddress string) (string, error) {
	_, err := cg.G.ExecCommand([]string{"ocr2:deploy", fmt.Sprintf("--minSubmissionValue=%d", minSubmissionValue), fmt.Sprintf("--maxSubmissionValue=%d", maxSubmissionValue), fmt.Sprintf("--decimals=%d", decimals), fmt.Sprintf("--name=%s", name), fmt.Sprintf("--link=%s", linkTokenAddress)}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Responses[0].Contract, nil
}

func (cg *CosmosGauntlet) DeployAccessControllerContract() (string, error) {
	_, err := cg.G.ExecCommand([]string{"access_controller:deploy"}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Responses[0].Contract, nil
}

func (cg *CosmosGauntlet) DeployOCR2ProxyContract(aggregator string) (string, error) {
	_, err := cg.G.ExecCommand([]string{"proxy_ocr2:deploy", aggregator}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Responses[0].Contract, nil
}

func (cg *CosmosGauntlet) SetOCRBilling(observationPaymentGjuels int64, transmissionPaymentGjuels int64, recommendedGasPriceMicro string, ocrAddress string) (string, error) {
	_, err := cg.G.ExecCommand([]string{"ocr2:set_billing", fmt.Sprintf("--observationPaymentGjuels=%d", observationPaymentGjuels), fmt.Sprintf("--transmissionPaymentGjuels=%d", transmissionPaymentGjuels), fmt.Sprintf("--recommendedGasPriceMicro=%s", recommendedGasPriceMicro), ocrAddress}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Responses[0].Contract, nil
}

func (cg *CosmosGauntlet) BeginProposal(ocrAddress string) (string, error) {
	_, err := cg.G.ExecCommand([]string{"ocr2:begin_proposal", ocrAddress}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Data["proposalId"].(string), nil
}

func (cg *CosmosGauntlet) ProposeConfig(cfg string, ocrAddress string) (string, error) {
	_, err := cg.G.ExecCommand([]string{"ocr2:propose_config", "--input=" + cfg, ocrAddress}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Responses[0].Contract, nil
}

func (cg *CosmosGauntlet) ProposeOffchainConfig(cfg string, ocrAddress string) (string, error) {
	_, err := cg.G.ExecCommand([]string{"ocr2:propose_offchain_config", "--input=" + cfg, ocrAddress}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Responses[0].Contract, nil
}

func (cg *CosmosGauntlet) FinalizeProposal(proposalId string, ocrAddress string) (string, error) {
	_, err := cg.G.ExecCommand([]string{"ocr2:finalize_proposal", "--proposalId=" + proposalId, ocrAddress}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Data["digest"].(string), nil
}

func (cg *CosmosGauntlet) AcceptProposal(cfg string, ocrAddress string) (string, error) {
	_, err := cg.G.ExecCommand([]string{"ocr2:accept_proposal", "--input=" + cfg, ocrAddress}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Responses[0].Contract, nil
}

func (cg *CosmosGauntlet) AddOCR2Access(aggregator, address string) (string, error) {
	_, err := cg.G.ExecCommand([]string{"ocr2:add_access", fmt.Sprintf("--address=%s", address), aggregator}, *cg.options)
	if err != nil {
		return "", err
	}
	cg.gr, err = cg.FetchGauntletJsonOutput()
	if err != nil {
		return "", err
	}
	return cg.gr.Responses[0].Contract, nil
}
