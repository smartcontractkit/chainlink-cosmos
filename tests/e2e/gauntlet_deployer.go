package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/utils"
	"github.com/smartcontractkit/integrations-framework/gauntlet"
)

const TERRA_COMMAND_ERROR = "Terra Command execution error"
const RETRY_COUNT = 5

type GauntletDeployer struct {
	Cli                        *gauntlet.Gauntlet
	Version                    string
	LinkToken                  string
	BillingAccessController    string
	RequesterAccessController  string
	Flags                      string
	DeviationFlaggingValidator string
	OCR                        string
	RddPath                    string
	ProposalId                 string
	ProposalDigest             string
	OffchainProposalSecret     string
}

type InspectionResult struct {
	Pass     bool
	Key      string
	Expected string
	Actual   string
}

// GetDefaultGauntletConfig gets the default config gauntlet will need to start making commands
// 	against the environment
func GetDefaultGauntletConfig(nodeUrl *url.URL) map[string]string {
	networkConfig := map[string]string{
		"NODE_URL":          nodeUrl.String(),
		"CHAIN_ID":          "localterra",
		"DEFAULT_GAS_PRICE": "1",
		"MNEMONIC":          "symbol force gallery make bulk round subway violin worry mixture penalty kingdom boring survey tool fringe patrol sausage hard admit remember broken alien absorb",
	}

	return networkConfig
}

// UpdateReportName updates the report name to be used by gauntlet on completion
func UpdateReportName(reportName string, g *gauntlet.Gauntlet) {
	g.NetworkConfig["REPORT_NAME"] = filepath.Join(utils.Reports, reportName)
	err := g.WriteNetworkConfigMap(utils.Networks)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to write the updated .env file")
}

// GetInspectionResultsFromOutput parses the inpsectiond data from the output
//  TODO we should really update the inspection command to just output json in the future
func GetInspectionResultsFromOutput(output string) (map[string]InspectionResult, error) {
	lines := strings.Split(output, "\n")
	passRegex, err := regexp.Compile("✅  (.+) matches: (.+)$")
	if err != nil {
		return map[string]InspectionResult{}, err
	}
	failRegex, err := regexp.Compile("⚠️   (.+) invalid: expected (.+) but actually (.*)$")
	if err != nil {
		return map[string]InspectionResult{}, err
	}
	results := map[string]InspectionResult{}
	for _, l := range lines {
		passMatches := passRegex.FindStringSubmatch(l)
		failMatches := failRegex.FindStringSubmatch(l)
		if len(passMatches) == 3 {
			results[passMatches[1]] = InspectionResult{
				Pass:     true,
				Key:      passMatches[1],
				Expected: "",
				Actual:   passMatches[2],
			}
		} else if len(failMatches) == 4 {
			results[failMatches[1]] = InspectionResult{
				Pass:     false,
				Key:      failMatches[1],
				Expected: failMatches[2],
				Actual:   failMatches[3],
			}
		}
	}

	return results, nil
}

// LoadReportJson loads a gauntlet report into a generic map
func LoadReportJson(file string) (map[string]interface{}, error) {
	jsonFile, err := os.Open(filepath.Join(utils.Reports, file))
	if err != nil {
		return map[string]interface{}{}, err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return map[string]interface{}{}, err
	}

	var data map[string]interface{}
	err = json.Unmarshal([]byte(byteValue), &data)

	return data, err
}

// GetTxAddressFromReport gets the address from the typical place in the json report data
func GetTxAddressFromReport(report map[string]interface{}) string {
	return report["responses"].([]interface{})[0].(map[string]interface{})["tx"].(map[string]interface{})["address"].(string)
}

// DeployToken deploys the link token
func (gd *GauntletDeployer) DeployToken() string {
	codeIds := gd.Cli.Flag("codeIDs", filepath.Join(utils.CodeIds, fmt.Sprintf("%s%s", gd.Cli.Network, ".json")))
	artifacts := gd.Cli.Flag("artifacts", filepath.Join(utils.GauntletTerraContracts, "artifacts", "bin"))
	reportName := "deploy_token"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"token:deploy",
		gd.Cli.Flag("version", gd.Version),
		codeIds,
		artifacts,
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to deploy link token")
	report, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	return GetTxAddressFromReport(report)
}

// Upload uploads the terra contracts
func (gd *GauntletDeployer) Upload() {
	UpdateReportName("upload", gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"upload",
		gd.Cli.Flag("version", gd.Version),
		gd.Cli.Flag("maxRetry", "10"),
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to upload contracts")
}

// deployAccessController deploys an access controller
func (gd *GauntletDeployer) deployAccessController(name string) string {
	codeIds := gd.Cli.Flag("codeIDs", filepath.Join(utils.CodeIds, fmt.Sprintf("%s%s", gd.Cli.Network, ".json")))
	UpdateReportName(name, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"access_controller:deploy",
		gd.Cli.Flag("version", gd.Version),
		codeIds,
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to deploy the billing access controller")
	report, err := LoadReportJson(name + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	return GetTxAddressFromReport(report)
}

// DeployBillingAccessController deploys a biller
func (gd *GauntletDeployer) DeployBillingAccessController() string {
	billingAccessController := gd.deployAccessController("billing_ac_deploy")
	gd.Cli.NetworkConfig["BILLING_ACCESS_CONTROLLER"] = billingAccessController
	return billingAccessController
}

// DeployRequesterAccessController deploys a requester
func (gd *GauntletDeployer) DeployRequesterAccessController() string {
	requesterAccessController := gd.deployAccessController("requester_ac_deploy")
	gd.Cli.NetworkConfig["REQUESTER_ACCESS_CONTROLLER"] = requesterAccessController
	return requesterAccessController
}

// DeployFlags deploys the flags for the lowering and raising access controllers
func (gd *GauntletDeployer) DeployFlags(billingAccessController, requesterAccessController string) string {
	reportName := "flags_deploy"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"flags:deploy",
		gd.Cli.Flag("loweringAccessController", billingAccessController),
		gd.Cli.Flag("raisingAccessController", requesterAccessController),
		gd.Cli.Flag("version", gd.Version),
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to deploy the flag")
	flagsReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	flags := GetTxAddressFromReport(flagsReport)
	return flags
}

// DeployDeviationFlaggingValidator deploys the deviation flagging validator with the threshold provided
func (gd *GauntletDeployer) DeployDeviationFlaggingValidator(flags string, flaggingThreshold int) string {
	reportName := "dfv_deploy"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"deviation_flagging_validator:deploy",
		gd.Cli.Flag("flaggingThreshold", fmt.Sprintf("%v", uint32(flaggingThreshold))),
		gd.Cli.Flag("flags", flags),
		gd.Cli.Flag("version", gd.Version),
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to deploy the deviation flagging validator")
	dfvReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	return GetTxAddressFromReport(dfvReport)
}

// DeployOcr deploys ocr, it creates an rdd file in the process and updates it with the ocr address on completion
func (gd *GauntletDeployer) DeployOcr() (string, string) {
	rddPath := filepath.Join(utils.Rdd, fmt.Sprintf("directory-terra-%s.json", gd.Cli.Network))
	tmpId := "terra1test0000000000000000000000000000000000"
	ocrRddContract := NewRddContract(tmpId)
	err := WriteRdd(ocrRddContract, rddPath)
	Expect(err).ShouldNot(HaveOccurred(), "Did not write the rdd json correctly")
	reportName := "ocr_deploy"
	UpdateReportName(reportName, gd.Cli)
	_, err = gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:deploy",
		gd.Cli.Flag("rdd", rddPath),
		gd.Cli.Flag("version", gd.Version),
		tmpId,
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to deploy ocr2")
	ocrReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	ocr := GetTxAddressFromReport(ocrReport)

	// add the new contract to the rdd
	ocrRddContract.Contracts[ocr] = ocrRddContract.Contracts[tmpId]
	err = WriteRdd(ocrRddContract, rddPath)
	Expect(err).ShouldNot(HaveOccurred(), "Did not write the rdd json correctly")
	return ocr, rddPath
}

// SetBiling sets the billing info that exists in the rdd file for the ocr address you pass in
func (gd *GauntletDeployer) SetBilling(ocr, rddPath string) {
	UpdateReportName("set_billing", gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:set_billing",
		gd.Cli.Flag("version", gd.Version),
		gd.Cli.Flag("rdd", rddPath),
		ocr,
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to set billing")
}

// BeginProposal begins the proposal
func (gd *GauntletDeployer) BeginProposal(ocr, rddPath string) string {
	reportName := "begin_proposal"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:begin_proposal",
		gd.Cli.Flag("version", gd.Version),
		gd.Cli.Flag("rdd", rddPath),
		ocr,
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to begin proposal")
	beginProposalReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	return beginProposalReport["data"].(map[string]interface{})["proposalId"].(string)
}

// ProposeConfig proposes the config
func (gd *GauntletDeployer) ProposeConfig(ocr, proposalId, rddPath string) {
	reportName := "propose_config"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:propose_config",
		gd.Cli.Flag("version", gd.Version),
		gd.Cli.Flag("rdd", rddPath),
		gd.Cli.Flag("proposalId", proposalId),
		gd.OCR,
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to propose config")
}

// ProposeOffchainConfig proposes the offchain config
func (gd *GauntletDeployer) ProposeOffchainConfig(ocr, proposalId, rddPath string) string {
	reportName := "propose_offchain_config"
	gd.Cli.NetworkConfig["SECRET"] = gd.Cli.NetworkConfig["MNEMONIC"]
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:propose_offchain_config",
		gd.Cli.Flag("version", gd.Version),
		gd.Cli.Flag("rdd", rddPath),
		gd.Cli.Flag("proposalId", proposalId),
		ocr,
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to propose offchain config")
	offchainProposalReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	return offchainProposalReport["data"].(map[string]interface{})["secret"].(string)
}

// FinalizeProposal finalizes the proposal
func (gd *GauntletDeployer) FinalizeProposal(ocr, proposalId, rddPath string) string {
	reportName := "finalize_proposal"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:finalize_proposal",
		gd.Cli.Flag("version", gd.Version),
		gd.Cli.Flag("rdd", rddPath),
		gd.Cli.Flag("proposalId", proposalId),
		ocr,
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to finalize proposal")
	finalizeProposalReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	return finalizeProposalReport["data"].(map[string]interface{})["digest"].(string)
}

// AcceptProposal accepts the proposal
func (gd *GauntletDeployer) AcceptProposal(ocr, proposalId, proposalDigest, secret, rddPath string) string {
	reportName := "accept_proposal"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:accept_proposal",
		gd.Cli.Flag("version", gd.Version),
		gd.Cli.Flag("rdd", rddPath),
		gd.Cli.Flag("proposalId", proposalId),
		gd.Cli.Flag("digest", proposalDigest),
		gd.Cli.Flag("secret", secret),
		ocr,
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to accept proposal")
	acceptProposalReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	return acceptProposalReport["data"].(map[string]interface{})["digest"].(string)
}

// OcrInspect gets the inspections results data
func (gd *GauntletDeployer) OcrInspect(ocr, rddPath string) map[string]InspectionResult {
	UpdateReportName("inspect", gd.Cli)
	output, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:inspect",
		gd.Cli.Flag("version", gd.Version),
		gd.Cli.Flag("rdd", rddPath),
		ocr,
	}, gauntlet.ExecCommandOptions{
		ErrHandling: []string{TERRA_COMMAND_ERROR},
		RetryCount:  RETRY_COUNT,
	})
	Expect(err).ShouldNot(HaveOccurred(), "Failed to inspect")

	results, err := GetInspectionResultsFromOutput(output)
	Expect(err).ShouldNot(HaveOccurred())
	return results
}
