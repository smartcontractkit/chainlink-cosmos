package ocr2_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/common"
	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/gauntlet"
	"github.com/smartcontractkit/chainlink-cosmos/ops/utils"
	"github.com/smartcontractkit/chainlink/integration-tests/actions"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestOCRBasic(t *testing.T) {
	logger := common.GetTestLogger(t)
	commonConfig := common.NewCommon()

	gauntletWorkingDir := fmt.Sprintf("%s/", utils.ProjectRoot)
	logger.Info().Str("working dir", gauntletWorkingDir).Msg("Initializing gauntlet")

	cg, err := gauntlet.NewCosmosGauntlet(gauntletWorkingDir)
	require.NoError(t, err, "Could not create cosmos gauntlet")

	err = cg.InstallDependencies()
	require.NoError(t, err, "Failed to install gauntlet dependencies")

	err = cg.SetupNetwork(commonConfig.NodeUrl, commonConfig.Mnemonic)
	require.NoError(t, err, "Setting up gauntlet network should not fail")

	// TODO: uncomment once we are ready to test rounds
	// commonConfig.SetDefaultEnvironment(t)

	// TODO: fund nodes if necessary

	// store the cw20_base contract so we have the token contract, and then deploy the LINK token.
	_, err = cg.UploadContracts(nil)
	require.NoError(t, err, "Could not upload cw20_base contract")

	linkTokenAddress, err := cg.DeployLinkTokenContract()
	require.NoError(t, err, "Could not deploy link token contract")
	logger.Info().Str("address", linkTokenAddress).Msg("Deployed LINK token")
	os.Setenv("LINK", linkTokenAddress)

	accessControllerAddress, err := cg.DeployAccessControllerContract()
	require.NoError(t, err, "Could not deploy access controller")
	logger.Info().Str("address", accessControllerAddress).Msg("Deployed access controller")
	os.Setenv("BILLING_ACCESS_CONTROLLER", accessControllerAddress)

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

	_, err = cg.AddOCR2Access(ocrAddress, ocrProxyAddress)
	require.NoError(t, err, "Could not add OCR2 access")
	logger.Info().Msg("Added OCR2 access")

	_, err = cg.MintLinkToken(linkTokenAddress, ocrAddress, "100000000000000000000")
	require.NoError(t, err, "Could not mint LINK token")

	observationPaymentGjuels := int64(1)
	transmissionPaymentGjuels := int64(1)
	_, err = cg.SetOCRBilling(observationPaymentGjuels, transmissionPaymentGjuels, ocrAddress)
	require.NoError(t, err, "Could not set OCR billing")

	chainlinkClient, err := common.NewChainlinkClient(commonConfig.Env, commonConfig.ChainName, commonConfig.ChainId, commonConfig.NodeUrl)
	require.NoError(t, err, "Could not create chainlink client")

	cfg, err := chainlinkClient.LoadOCR2Config([]string{commonConfig.Account})
	require.NoError(t, err, "Could not load OCR2 config")

	var parsedConfig []byte
	parsedConfig, err = json.Marshal(cfg)
	require.NoError(t, err, "Could not parse JSON config")

	_, err = cg.SetConfigDetails(string(parsedConfig), ocrAddress)
	require.NoError(t, err, "Could not set config details")

	//if !testState.Common.Testnet {
	//testState.Devnet.AutoLoadState(testState.OCR2Client, testState.OCRAddr)
	//}
	//mockServerVal = 900000000
	//testState.SetUpNodes(mockServerVal)

	//err = testState.ValidateRounds(10, false)
	//require.NoError(t, err, "Validating round should not fail")

	t.Cleanup(func() {
		err = actions.TeardownSuite(t, commonConfig.Env, "./", nil, nil, zapcore.DPanicLevel, nil)
		// err = actions.TeardownSuite(t, testState.Common.Env, utils.ProjectRoot, testState.Cc.ChainlinkNodes, nil, zapcore.ErrorLevel)
		require.NoError(t, err, "Error tearing down environment")
	})
}
