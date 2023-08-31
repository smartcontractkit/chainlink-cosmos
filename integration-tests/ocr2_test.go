package ocr2_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/common"
	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/gauntlet"
	"github.com/smartcontractkit/chainlink-cosmos/ops/utils"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/adapters/cosmwasm"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/client"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/params"
	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/testutil"
	relaylogger "github.com/smartcontractkit/chainlink-relay/pkg/logger"

	// "github.com/smartcontractkit/chainlink/integration-tests/actions"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2/types"

	cometbfttypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	// "go.uber.org/zap/zapcore"
)

func TestOCRBasic(t *testing.T) {
	// Set up test environment
	logger := common.GetTestLogger(t)
	commonConfig := common.NewCommon(t)
	commonConfig.SetLocalEnvironment(t)

	bech32Prefix := "wasm"
	params.InitCosmosSdk(
		bech32Prefix,
		/* token= */ "cosm",
	)
	clientLogger, err := relaylogger.New()
	require.NoError(t, err, "Could not create relay logger")
	cosmosClient, err := client.NewClient(
		commonConfig.ChainId,
		commonConfig.NodeUrl,
		30*time.Second,
		clientLogger)
	require.NoError(t, err, "Could not create cosmos client")

	nodeName := "primary"
	chainlinkClient, err := common.NewChainlinkClient(commonConfig.Env, commonConfig.ChainId, nodeName, commonConfig.NodeUrl, bech32Prefix)
	require.NoError(t, err, "Could not create chainlink client")

	logger.Info().Str("node addresses", strings.Join(chainlinkClient.GetNodeAddresses(), " ")).Msg("Created chainlink client")
	privateKey, testAccount, err := testutil.CreateKeyFromMnemonic(commonConfig.Mnemonic)
	require.NoError(t, err, "Could not create private key from mnemonic")
	logger.Info().Str("from", testAccount.String()).Msg("Funding nodes")

	gasPrice := types.NewDecCoinFromDec("ucosm", types.MustNewDecFromStr("1"))
	amount := []types.Coin{types.NewCoin("ucosm", types.NewInt(int64(10000000)))}
	accountNumber, sequenceNumber, err := cosmosClient.Account(testAccount)
	require.NoError(t, err, "Could not get account")

	for i, nodeAddr := range chainlinkClient.GetNodeAddresses() {
		to := types.MustAccAddressFromBech32(nodeAddr)
		msgSend := banktypes.NewMsgSend(testAccount, to, amount)
		resp, err := cosmosClient.SignAndBroadcast([]types.Msg{msgSend}, accountNumber, sequenceNumber+uint64(i), gasPrice, privateKey, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
		require.NoError(t, err, "Could not send tokens")
		logger.Info().Str("from", testAccount.String()).
			Str("to", nodeAddr).
			Str("amount", "10000000").
			Str("token", "ucosm").
			Str("txHash", resp.TxResponse.TxHash).
			Msg("Sending native tokens")
		tx, success := client.AwaitTxCommitted(t, cosmosClient, resp.TxResponse.TxHash)
		require.True(t, success)
		require.Equal(t, cometbfttypes.CodeTypeOK, tx.TxResponse.Code)
		balance, err := cosmosClient.Balance(to, "ucosm")
		require.NoError(t, err, "Could not fetch ucosm balance")
		require.Equal(t, balance.String(), "10000000ucosm")
	}

	gauntletWorkingDir := fmt.Sprintf("%s/", utils.ProjectRoot)
	logger.Info().Str("working dir", gauntletWorkingDir).Msg("Initializing gauntlet")

	cg, err := gauntlet.NewCosmosGauntlet(gauntletWorkingDir)
	require.NoError(t, err, "Could not create cosmos gauntlet")

	err = cg.InstallDependencies()
	require.NoError(t, err, "Failed to install gauntlet dependencies")

	err = cg.SetupNetwork(commonConfig.NodeUrl, commonConfig.Mnemonic)
	require.NoError(t, err, "Setting up gauntlet network should not fail")

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

	cfg, err := chainlinkClient.LoadOCR2Config(proposalId)
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

	p2pPort := "50200"
	err = chainlinkClient.CreateJobsForContract(
		commonConfig.ChainId,
		nodeName,
		p2pPort,
		commonConfig.MockUrl,
		commonConfig.JuelsPerFeeCoinSource,
		ocrAddress)
	require.NoError(t, err, "Could not create jobs for contract")

	err = validateRounds(t, cosmosClient, types.MustAccAddressFromBech32(ocrAddress), types.MustAccAddressFromBech32(ocrProxyAddress), commonConfig.IsSoak, commonConfig.TestDuration)
	require.NoError(t, err, "Validating round should not fail")

	// Tear down local stack
	commonConfig.TearDownLocalEnvironment(t)

	// t.Cleanup(func() {
	// 	err = actions.TeardownSuite(t, commonConfig.Env, "./", nil, nil, zapcore.DPanicLevel, nil)
	// 	//err = actions.TeardownSuite(t, t.Common.Env, utils.ProjectRoot, t.Cc.ChainlinkNodes, nil, zapcore.ErrorLevel)
	// 	require.NoError(t, err, "Error tearing down environment")
	// })
}

func validateRounds(t *testing.T, cosmosClient *client.Client, ocrAddress types.AccAddress, ocrProxyAddress types.AccAddress, isSoak bool, testDuration time.Duration) error {
	var rounds int
	if isSoak {
		rounds = 99999999
	} else {
		rounds = 10
	}

	// TODO(BCI-1746): dynamic mock-adapter values
	mockAdapterValue := 5

	logger := common.GetTestLogger(t)
	ctx := context.Background() // context background used because timeout handled by requestTimeout param
	// assert new rounds are occurring
	increasing := 0 // track number of increasing rounds
	var stuck bool
	stuckCount := 0
	var positive bool
	resp, err := cosmosClient.ContractState(
		ocrAddress,
		[]byte(`{"link_available_for_payment":{}}`),
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
	logger.Info().Str("amount", linkResponse.Amount).Msg("Queried link available for payment")

	availableLink, success := new(big.Int).SetString(linkResponse.Amount, 10)
	require.True(t, success, "Could not convert link_available_for_payment response")
	require.True(t, availableLink.Cmp(big.NewInt(0)) > 0, "Aggregator should have non-zero balance")

	// TODO(BCI-1767): this needs to be able to support different readers
	ocrLogger, err := relaylogger.New()
	require.NoError(t, err, "Failed to create OCR relay logger")
	ocrReader := cosmwasm.NewOCR2Reader(ocrAddress, cosmosClient, ocrLogger)

	type TransmissionDetails struct {
		ConfigDigest    ocrtypes.ConfigDigest
		Epoch           uint32
		Round           uint8
		LatestAnswer    *big.Int
		LatestTimestamp time.Time
	}

	previous := TransmissionDetails{}

	for start := time.Now(); time.Since(start) < testDuration; {
		logger.Info().Msg(fmt.Sprintf("Elapsed time: %s, Round wait: %s ", time.Since(start), testDuration))
		configDigest, epoch, round, latestAnswer, latestTimestamp, err := ocrReader.LatestTransmissionDetails(ctx)
		require.NoError(t, err, "Failed to get latest transmission details")
		// end condition: enough rounds have occurred
		if !isSoak && increasing >= rounds && positive {
			break
		}

		// end condition: rounds have been stuck
		if stuck && stuckCount > 50 {
			logger.Debug().Msg("failing to fetch transmissions means blockchain may have stopped")
			break
		}

		// try to fetch rounds
		time.Sleep(5 * time.Second)

		if err != nil {
			logger.Error().Msg(fmt.Sprintf("Transmission Error: %+v", err))
			continue
		}
		logger.Info().Msg(fmt.Sprintf("Transmission Details: configDigest: %+v, epoch: %+v, round: %+v, latestAnswer: %+v, latestTimestamp: %+v", configDigest, epoch, round, latestAnswer, latestTimestamp))

		// continue if no changes
		if epoch == 0 && round == 0 {
			continue
		}

		ansCmp := latestAnswer.Cmp(big.NewInt(0))
		positive = ansCmp == 1 || positive

		// if changes from zero values set (should only initially)
		if epoch > 0 && previous.Epoch == 0 {
			if !isSoak {
				require.Greater(t, epoch, previous.Epoch)
				require.GreaterOrEqual(t, round, previous.Round)
				require.NotEqual(t, ansCmp, 0) // require changed from 0
				require.NotEqual(t, configDigest, previous.ConfigDigest)
				require.Equal(t, previous.LatestTimestamp.Before(latestTimestamp), true)
			}
			previous = TransmissionDetails{
				ConfigDigest:    configDigest,
				Epoch:           epoch,
				Round:           round,
				LatestAnswer:    latestAnswer,
				LatestTimestamp: latestTimestamp,
			}
			continue
		}
		// check increasing rounds
		if !isSoak {
			require.Equal(t, configDigest, previous.ConfigDigest, "Config digest should not change")
		} else {
			if configDigest != previous.ConfigDigest {
				logger.Error().Msg(fmt.Sprintf("Config digest should not change, expected %s got %s", previous.ConfigDigest, configDigest))
			}
		}
		if (epoch > previous.Epoch || (epoch == previous.Epoch && round > previous.Round)) && previous.LatestTimestamp.Before(latestTimestamp) {
			increasing++
			stuck = false
			stuckCount = 0 // reset counter
			continue
		}

		// reach this point, answer has not changed
		stuckCount++
		if stuckCount > 30 {
			stuck = true
			increasing = 0
		}
	}
	if !isSoak {
		require.GreaterOrEqual(t, increasing, rounds, "Round + epochs should be increasing")
		require.Equal(t, positive, true, "Positive value should have been submitted")
		require.Equal(t, stuck, false, "Round + epochs should not be stuck")
	}

	// Test proxy reading
	// TODO: would be good to test proxy switching underlying feeds
	resp, err = cosmosClient.ContractState(ocrProxyAddress, []byte(`{"latest_round_data":{}}`))
	if !isSoak {
		require.NoError(t, err, "Reading round data from proxy should not fail")
		//assert.Equal(t, len(roundDataRaw), 5, "Round data from proxy should match expected size")
	}
	roundData := struct {
		Answer string `json:"answer"`
	}{}
	err = json.Unmarshal(resp, &roundData)
	require.NoError(t, err, "Failed to unmarshal round data")

	valueBig, success := new(big.Int).SetString(roundData.Answer, 10)
	require.True(t, success, "Failed to parse round data")
	value := valueBig.Int64()
	require.Equal(t, value, int64(mockAdapterValue), "Reading from proxy should return correct value")

	return nil
}
