package ocr2_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/common"
	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/gauntlet"
	"github.com/smartcontractkit/chainlink-cosmos/ops/utils"

	"github.com/stretchr/testify/require"
)

func TestOCRBasic(t *testing.T) {
	// Set up test environment
	logger := common.GetTestLogger(t)
	commonConfig := common.NewCommon(t)
	commonConfig.SetLocalEnvironment()

	chainlinkClient, err := common.NewChainlinkClient(commonConfig.Env, commonConfig.ChainName, commonConfig.ChainId, commonConfig.NodeUrl)
	require.NoError(t, err, "Could not create chainlink client")

	gauntletWorkingDir := fmt.Sprintf("%s/", utils.ProjectRoot)
	logger.Info().Str("working dir", gauntletWorkingDir).Msg("Initializing gauntlet")

	cg, err := gauntlet.NewCosmosGauntlet(gauntletWorkingDir)
	require.NoError(t, err, "Could not create cosmos gauntlet")

	err = cg.InstallDependencies()
	require.NoError(t, err, "Failed to install gauntlet dependencies")

	err = cg.SetupNetwork(commonConfig.NodeUrl, commonConfig.Mnemonic)
	require.NoError(t, err, "Setting up gauntlet network should not fail")

	// TODO: fund nodes if necessary

	// Upload contracts
	_, err = cg.UploadContracts(nil)
	require.NoError(t, err, "Could not upload contracts")

	// Deploy contracts
	linkTokenAddress, err := cg.DeployLinkTokenContract()
	require.NoError(t, err, "Could not deploy link token contract")
	logger.Info().Str("address", linkTokenAddress).Msg("Deployed LINK token")
	os.Setenv("LINK", linkTokenAddress)

	billingAccessControllerAddress, err := cg.DeployAccessControllerContract()
	require.NoError(t, err, "Could not deploy billing access controller")
	logger.Info().Str("address", billingAccessControllerAddress).Msg("Deployed billing access controller")
	os.Setenv("BILLING_ACCESS_CONTROLLER", billingAccessControllerAddress)

	requesterAccessControllerAddress, err := cg.DeployAccessControllerContract()
	require.NoError(t, err, "Could not deploy requester access controller")
	logger.Info().Str("address", requesterAccessControllerAddress).Msg("Deployed requester access controller")
	os.Setenv("REQUESTER_ACCESS_CONTROLLER", requesterAccessControllerAddress)

	minSubmissionValue := int64(0)
	maxSubmissionValue := int64(100000000000)
	decimals := 9
	name := "auto"
	ocrAddress, err := cg.DeployOCR2ControllerContract(minSubmissionValue, maxSubmissionValue, decimals, name, linkTokenAddress)
	require.NoError(t, err, "Could not deploy OCR2 controller contract")
	logger.Info().Str("address", ocrAddress).Msg("Deployed OCR2 Controller contract")

	ocrProxyAddress, err := cg.DeployOCR2ProxyContract(ocrAddress)
	require.NoError(t, err, "Could not deploy OCR2 proxy contract")
	logger.Info().Str("address", ocrProxyAddress).Msg("Deployed OCR2 proxy contract")

	// Mint LINK tokens to aggregator
	_, err = cg.MintLinkToken(linkTokenAddress, ocrAddress, "100000000000000000000")
	require.NoError(t, err, "Could not mint LINK token")

	// Set OCR2 Billing
	observationPaymentGjuels := int64(1)
	transmissionPaymentGjuels := int64(1)
	recommendedGasPriceMicro := "1"
	_, err = cg.SetOCRBilling(observationPaymentGjuels, transmissionPaymentGjuels, recommendedGasPriceMicro, ocrAddress)
	require.NoError(t, err, "Could not set OCR billing")

	// OCR2 Config Proposal
	proposalId, err := cg.BeginProposal(ocrAddress)
	require.NoError(t, err, "Could not begin proposal")

	cfg, err := chainlinkClient.LoadOCR2Config()
	require.NoError(t, err, "Could not load OCR2 config")
	cfg.ProposalId = proposalId
	cfg.Payees = cfg.Transmitters // Set payees to same addresses as transmitters

	var parsedConfig []byte
	parsedConfig, err = json.Marshal(cfg)
	require.NoError(t, err, "Could not parse JSON config")

	_, err = cg.ProposeConfig(string(parsedConfig), ocrAddress)
	require.NoError(t, err, "Could not propose config")

	_, err = cg.ProposeOffchainConfig(string(parsedConfig), ocrAddress)
	require.NoError(t, err, "Could not propose offchain config")

	digest, err := cg.FinalizeProposal(proposalId, ocrAddress)
	require.NoError(t, err, "Could not finalize proposal")

	var acceptProposalInput = struct {
		ProposalId     string            `json:"proposalId"`
		Digest         string            `json:"digest"`
		OffchainConfig common.OCR2Config `json:"offchainConfig"`
		RandomSecret   string            `json:"randomSecret"`
	}{
		ProposalId:     proposalId,
		Digest:         digest,
		OffchainConfig: *cfg,
		RandomSecret:   cfg.Secret,
	}
	var parsedInput []byte
	parsedInput, err = json.Marshal(acceptProposalInput)
	require.NoError(t, err, "Could not parse JSON input")
	_, err = cg.AcceptProposal(string(parsedInput), ocrAddress)
	require.NoError(t, err, "Could not accept proposed config")

	//if !testState.Common.Testnet {
	//testState.Devnet.AutoLoadState(testState.OCR2Client, testState.OCRAddr)
	//}
	//mockServerVal = 900000000

	//testState.SetUpNodes(mockServerVal)

	//err = testState.ValidateRounds(10, false)
	//require.NoError(t, err, "Validating round should not fail")

	// t.Cleanup(func() {
	// 	// err = actions.TeardownSuite(t, commonConfig.Env, "./", nil, nil, zapcore.DPanicLevel, nil)
	// 	err = actions.TeardownSuite(t, t.Common.Env, utils.ProjectRoot, t.Cc.ChainlinkNodes, nil, zapcore.ErrorLevel)
	// 	require.NoError(t, err, "Error tearing down environment")
	// })
}
