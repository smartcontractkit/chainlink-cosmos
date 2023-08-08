package ocr2_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/common"
	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/gauntlet"
	"github.com/smartcontractkit/chainlink-cosmos/ops/utils"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/client"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/params"
	relaylogger "github.com/smartcontractkit/chainlink-relay/pkg/logger"
	"github.com/smartcontractkit/chainlink/integration-tests/actions"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestOCRBasic(t *testing.T) {
	// Set up test environment
	logger := common.GetTestLogger(t)
	commonConfig := common.NewCommon(t)
	commonConfig.SetLocalEnvironment()

	chainlinkClient, err := common.NewChainlinkClient(commonConfig.Env, commonConfig.ChainName, commonConfig.ChainId, commonConfig.NodeUrl)
	require.NoError(t, err, "Could not create chainlink client")

	params.InitCosmosSdk(
		/* bech32Prefix= */ "wasm",
		/* token= */ "atom",
	)
	clientLogger, err := relaylogger.New()
	require.NoError(t, err, "Could not create relay logger")
	cosmosClient, err := client.NewClient(
		commonConfig.ChainId,
		commonConfig.NodeUrl,
		30*time.Second,
		clientLogger)
	require.NoError(t, err, "Could not create cosmos client")

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

	cfg, err := chainlinkClient.LoadOCR2Config(proposalId, []string{commonConfig.Account})
	require.NoError(t, err, "Could not load OCR2 config")

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

	p2pPort := "6690"

	err = chainlinkClient.CreateJobsForContract(
		commonConfig.ChainId,
		p2pPort,
		commonConfig.MockUrl,
		commonConfig.ObservationSource,
		commonConfig.JuelsPerFeeCoinSource,
		ocrAddress)
	require.NoError(t, err, "Could not create jobs for contract")

	err = validateRounds(t, cosmosClient, ocrAddress, commonConfig.IsSoak)
	require.NoError(t, err, "Validating round should not fail")

	//if !testState.Common.Testnet {
	//testState.Devnet.AutoLoadState(testState.OCR2Client, testState.OCRAddr)
	//}
	//mockServerVal = 900000000

	//testState.SetUpNodes(mockServerVal)

	//err = testState.ValidateRounds(10, false)
	//require.NoError(t, err, "Validating round should not fail")

	t.Cleanup(func() {
		err = actions.TeardownSuite(t, commonConfig.Env, "./", nil, nil, zapcore.DPanicLevel, nil)
		//err = actions.TeardownSuite(t, t.Common.Env, utils.ProjectRoot, t.Cc.ChainlinkNodes, nil, zapcore.ErrorLevel)
		require.NoError(t, err, "Error tearing down environment")
	})
}

func validateRounds(t *testing.T, cosmosClient *client.Client, ocrAddress string, isSoak bool) error {
	logger := common.GetTestLogger(t)
	//ctx := context.Background() // context background used because timeout handled by requestTimeout param
	// assert new rounds are occurring
	//details := ocr2.TransmissionDetails{}
	//increasing := 0 // track number of increasing rounds
	//var stuck bool
	//stuckCount := 0
	//var positive bool
	resp, err := cosmosClient.ContractState(
		types.MustAccAddressFromBech32(ocrAddress),
		[]byte(`"link_available_for_payment"`),
	)
	if err != nil {
		return err
	}

	linkResponse := struct {
		Amount string `json:"amount"`
	}{}
	if err := json.Unmarshal(resp, &linkResponse); err != nil {
		return err
	}
	availableLink, err := strconv.ParseInt(linkResponse.Amount, 10, 64)
	require.NoError(t, err, "Could not convert link_available_for_payment response")
	logger.Info().Int64("available", availableLink).Msg("Queried link available for payment")
	require.GreaterOrEqual(t, availableLink, 1, "Aggregator should have non-zero balance")

	//// validate balance in aggregator
	//resLINK, errLINK := testState.Starknet.CallContract(ctx, starknet.CallOps{
	//ContractAddress: caigotypes.StrToFelt(testState.LinkTokenAddr),
	//Selector:        "balance_of",
	//Calldata:        []string{testState.OCRAddr},
	//})
	//require.NoError(testState.T, errLINK, "Reader balance from LINK contract should not fail")
	//resAgg, errAgg := testState.Starknet.CallContract(ctx, starknet.CallOps{
	//ContractAddress: caigotypes.StrToFelt(testState.OCRAddr),
	//Selector:        "link_available_for_payment",
	//})
	//require.NoError(testState.T, errAgg, "Reader balance from LINK contract should not fail")
	//balLINK, _ := new(big.Int).SetString(resLINK[0], 0)
	//balAgg, _ := new(big.Int).SetString(resAgg[1], 0)
	//isNegative, _ := new(big.Int).SetString(resAgg[0], 0)
	//if isNegative.Sign() > 0 {
	//balAgg = new(big.Int).Neg(balAgg)
	//}

	//assert.Equal(testState.T, balLINK.Cmp(big.NewInt(0)), 1, "Aggregator should have non-zero balance")
	//assert.GreaterOrEqual(testState.T, balLINK.Cmp(balAgg), 0, "Aggregator payment balance should be <= actual LINK balance")

	//for start := time.Now(); time.Since(start) < testState.Common.TestDuration; {
	//l.Info().Msg(fmt.Sprintf("Elapsed time: %s, Round wait: %s ", time.Since(start), testState.Common.TestDuration))
	//res, err := testState.OCR2Client.LatestTransmissionDetails(ctx, caigotypes.StrToFelt(testState.OCRAddr))
	//require.NoError(testState.T, err, "Failed to get latest transmission details")
	//// end condition: enough rounds have occurred
	//if !isSoak && increasing >= rounds && positive {
	//break
	//}

	//// end condition: rounds have been stuck
	//if stuck && stuckCount > 50 {
	//l.Debug().Msg("failing to fetch transmissions means blockchain may have stopped")
	//break
	//}

	//l.Info().Msg(fmt.Sprintf("Setting adapter value to %d", mockServerValue))
	//err = testState.SetMockServerValue("", mockServerValue)
	//if err != nil {
	//l.Error().Msg(fmt.Sprintf("Setting mock server value error: %+v", err))
	//}
	//// try to fetch rounds
	//time.Sleep(5 * time.Second)

	//if err != nil {
	//l.Error().Msg(fmt.Sprintf("Transmission Error: %+v", err))
	//continue
	//}
	//l.Info().Msg(fmt.Sprintf("Transmission Details: %+v", res))

	//// continue if no changes
	//if res.Epoch == 0 && res.Round == 0 {
	//continue
	//}

	//ansCmp := res.LatestAnswer.Cmp(big.NewInt(0))
	//positive = ansCmp == 1 || positive

	//// if changes from zero values set (should only initially)
	//if res.Epoch > 0 && details.Epoch == 0 {
	//if !isSoak {
	//assert.Greater(testState.T, res.Epoch, details.Epoch)
	//assert.GreaterOrEqual(testState.T, res.Round, details.Round)
	//assert.NotEqual(testState.T, ansCmp, 0) // assert changed from 0
	//assert.NotEqual(testState.T, res.Digest, details.Digest)
	//assert.Equal(testState.T, details.LatestTimestamp.Before(res.LatestTimestamp), true)
	//}
	//details = res
	//continue
	//}
	//// check increasing rounds
	//if !isSoak {
	//assert.Equal(testState.T, res.Digest, details.Digest, "Config digest should not change")
	//} else {
	//if res.Digest != details.Digest {
	//l.Error().Msg(fmt.Sprintf("Config digest should not change, expected %s got %s", details.Digest, res.Digest))
	//}
	//}
	//if (res.Epoch > details.Epoch || (res.Epoch == details.Epoch && res.Round > details.Round)) && details.LatestTimestamp.Before(res.LatestTimestamp) {
	//increasing++
	//stuck = false
	//stuckCount = 0 // reset counter
	//continue
	//}

	//// reach this point, answer has not changed
	//stuckCount++
	//if stuckCount > 30 {
	//stuck = true
	//increasing = 0
	//}
	//}
	//if !isSoak {
	//assert.GreaterOrEqual(testState.T, increasing, rounds, "Round + epochs should be increasing")
	//assert.Equal(testState.T, positive, true, "Positive value should have been submitted")
	//assert.Equal(testState.T, stuck, false, "Round + epochs should not be stuck")
	//}

	//// Test proxy reading
	//// TODO: would be good to test proxy switching underlying feeds
	//roundDataRaw, err := testState.Starknet.CallContract(ctx, starknet.CallOps{
	//ContractAddress: caigotypes.StrToFelt(testState.ProxyAddr),
	//Selector:        "latest_round_data",
	//})
	//if !isSoak {
	//require.NoError(testState.T, err, "Reading round data from proxy should not fail")
	//assert.Equal(testState.T, len(roundDataRaw), 5, "Round data from proxy should match expected size")
	//}
	//valueBig, err := starknet.HexToUnsignedBig(roundDataRaw[1])
	//require.NoError(testState.T, err)
	//value := valueBig.Int64()
	//if value < 0 {
	//assert.Equal(testState.T, value, int64(mockServerValue), "Reading from proxy should return correct value")
	//}

	return nil
}
