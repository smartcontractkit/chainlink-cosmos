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

type GauntletDeployer struct {
	Cli                        *gauntlet.Gauntlet
	LinkToken                  string
	BillingAccessController    string
	RequesterAccessController  string
	Flags                      string
	DeviationFlaggingValidator string
	OCR                        string
	RddPath                    string
	ProposalId                 string
	ProposalDigest             string
}

type InspectionResult struct {
	Pass     bool
	Key      string
	Expected string
	Actual   string
}

// GetDefaultGauntletConfig gets  the default config gauntlet will need to start making commands
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

func UpdateReportName(reportName string, g *gauntlet.Gauntlet) {
	g.NetworkConfig["REPORT_NAME"] = filepath.Join(utils.Reports, reportName)
	err := g.WriteNetworkConfigMap(utils.Networks)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to write the updated .env file")
}

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

func GetTxAddressFromReport(report map[string]interface{}) string {
	return report["responses"].([]interface{})[0].(map[string]interface{})["tx"].(map[string]interface{})["address"].(string)
}

func (gd *GauntletDeployer) DeployToken() {
	// TODO figure out why this never passes??? for a future pr
	codeIds := gd.Cli.Flag("codeIDs", filepath.Join(utils.CodeIds, fmt.Sprintf("%s%s", gd.Cli.Network, ".json")))
	artifacts := gd.Cli.Flag("artifacts", filepath.Join(utils.ProjectRoot, "packages-ts/gauntlet-terra-contracts/artifacts/bin"))
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"token:deploy",
		gd.Cli.Flag("version", "local"),
		codeIds,
		artifacts,
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to deploy link token")
	// TODO parse link token and set into state when this is working
	//s.LinkToken = something
}

func (gd *GauntletDeployer) Upload() {
	UpdateReportName("upload", gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"upload",
		gd.Cli.Flag("version", "local"),
		gd.Cli.Flag("maxRetry", "10"),
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 5)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to upload contracts")
}

func (gd *GauntletDeployer) deployAccessController(name string) string {
	codeIds := gd.Cli.Flag("codeIDs", filepath.Join(utils.CodeIds, fmt.Sprintf("%s%s", gd.Cli.Network, ".json")))
	UpdateReportName(name, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"access_controller:deploy",
		gd.Cli.Flag("version", "local"),
		codeIds,
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to deploy the billing access controller")
	report, err := LoadReportJson(name + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	return GetTxAddressFromReport(report)
}

func (gd *GauntletDeployer) DeployBillingAccessController() {
	gd.BillingAccessController = gd.deployAccessController("billing_ac_deploy")
	gd.Cli.NetworkConfig["BILLING_ACCESS_CONTROLLER"] = gd.BillingAccessController
}

func (gd *GauntletDeployer) DeployRequesterAccessController() {
	gd.RequesterAccessController = gd.deployAccessController("requester_ac_deploy")
	gd.Cli.NetworkConfig["REQUESTER_ACCESS_CONTROLLER"] = gd.RequesterAccessController
}

func (gd *GauntletDeployer) DeployFlags() {
	reportName := "flags_deploy"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"flags:deploy",
		gd.Cli.Flag("loweringAccessController", gd.BillingAccessController),
		gd.Cli.Flag("raisingAccessController", gd.RequesterAccessController),
		gd.Cli.Flag("version", "local"),
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to deploy the flag")
	flagsReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	gd.Flags = GetTxAddressFromReport(flagsReport)
}

func (gd *GauntletDeployer) DeployDeviationFlaggingValidator() {
	reportName := "dfv_deploy"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"deviation_flagging_validator:deploy",
		gd.Cli.Flag("flaggingThreshold", fmt.Sprintf("%v", uint32(80000))),
		gd.Cli.Flag("flags", gd.Flags),
		gd.Cli.Flag("version", "local"),
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to deploy the deviation flagging validator")
	dfvReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	gd.DeviationFlaggingValidator = GetTxAddressFromReport(dfvReport)
}

func (gd *GauntletDeployer) DeployOcr() {
	gd.RddPath = filepath.Join("tests", "e2e", "smoke", "rdd", fmt.Sprintf("directory-terra-%s.json", gd.Cli.Network))
	tmpId := "terra1test0000000000000000000000000000000000"
	ocrRddContract := NewRddContract(tmpId)
	err := WriteRdd(ocrRddContract, gd.RddPath)
	Expect(err).ShouldNot(HaveOccurred(), "Did not write the rdd json correctly")
	reportName := "ocr_deploy"
	UpdateReportName(reportName, gd.Cli)
	_, err = gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:deploy",
		gd.Cli.Flag("rdd", gd.RddPath),
		gd.Cli.Flag("version", "local"),
		tmpId,
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to deploy ocr2")
	ocrReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	gd.OCR = GetTxAddressFromReport(ocrReport)

	// add the new contract to the rdd
	ocrRddContract.Contracts[gd.OCR] = ocrRddContract.Contracts[tmpId]
	err = WriteRdd(ocrRddContract, gd.RddPath)
	Expect(err).ShouldNot(HaveOccurred(), "Did not write the rdd json correctly")
}

func (gd *GauntletDeployer) SetBilling() {
	UpdateReportName("set_billing", gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:set_billing",
		gd.Cli.Flag("version", "local"),
		gd.Cli.Flag("rdd", gd.RddPath),
		gd.OCR,
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to set billing")
}

func (gd *GauntletDeployer) BeginProposal() {
	reportName := "begin_proposal"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:begin_proposal",
		gd.Cli.Flag("version", "local"),
		gd.Cli.Flag("rdd", gd.RddPath),
		gd.OCR,
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to begin proposal")
	beginProposalReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	gd.ProposalId = beginProposalReport["data"].(map[string]interface{})["proposalId"].(string)
}

func (gd *GauntletDeployer) ProposeConfig() {
	reportName := "propose_config"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:propose_config",
		gd.Cli.Flag("version", "local"),
		gd.Cli.Flag("rdd", gd.RddPath),
		gd.Cli.Flag("f", "1"),
		gd.Cli.Flag("proposalId", gd.ProposalId),
		gd.OCR,
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to propose config")
}

func (gd *GauntletDeployer) ProposeOffchainConfig() {
	reportName := "propose_offchain_config"
	gd.Cli.NetworkConfig["SECRET"] = gd.Cli.NetworkConfig["MNEMONIC"]
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:propose_offchain_config",
		gd.Cli.Flag("version", "local"),
		gd.Cli.Flag("rdd", gd.RddPath),
		gd.Cli.Flag("proposalId", gd.ProposalId),
		gd.Cli.Flag("offchainConfigVersion", "2"),
		gd.OCR,
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to propose offchain config")
}

func (gd *GauntletDeployer) FinalizeProposal() {
	reportName := "finalize_proposal"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:finalize_proposal",
		gd.Cli.Flag("version", "local"),
		gd.Cli.Flag("rdd", gd.RddPath),
		gd.Cli.Flag("proposalId", gd.ProposalId),
		gd.OCR,
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to finalize proposal")
	finalizeProposalReport, err := LoadReportJson(reportName + ".json")
	Expect(err).ShouldNot(HaveOccurred())
	gd.ProposalDigest = finalizeProposalReport["data"].(map[string]interface{})["digest"].(string)
}

func (gd *GauntletDeployer) AcceptProposal() {
	reportName := "accept_proposal"
	UpdateReportName(reportName, gd.Cli)
	_, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:accept_proposal",
		gd.Cli.Flag("version", "local"),
		gd.Cli.Flag("rdd", gd.RddPath),
		gd.Cli.Flag("proposalId", gd.ProposalId),
		gd.Cli.Flag("digest", gd.ProposalDigest),
		gd.OCR,
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to accept proposal")
}

func (gd *GauntletDeployer) OcrInspect() map[string]InspectionResult {
	UpdateReportName("inspect", gd.Cli)
	output, err := gd.Cli.ExecCommandWithRetries([]string{
		"ocr2:inspect",
		gd.Cli.Flag("version", "local"),
		gd.Cli.Flag("rdd", gd.RddPath),
		gd.OCR,
	}, []string{
		TERRA_COMMAND_ERROR,
	}, 2)
	Expect(err).ShouldNot(HaveOccurred(), "Failed to inspect")

	results, err := GetInspectionResultsFromOutput(output)
	Expect(err).ShouldNot(HaveOccurred())
	return results
}
