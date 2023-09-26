package ocr2_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
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
	"github.com/smartcontractkit/chainlink-cosmos/pkg/monitoring"
	relaylogger "github.com/smartcontractkit/chainlink-relay/pkg/logger"

	// "github.com/smartcontractkit/chainlink/integration-tests/actions"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2/types"

	cometbfttypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
	// "go.uber.org/zap/zapcore"
)

func TestOCRBasic(t *testing.T) {
	// TODO: set back to false
	runOcrTest(t, true)
}

func runOcrTest(t *testing.T, useMonitor bool) {
	// Set up test environment
	logger := common.GetTestLogger(t)
	commonConfig := common.NewCommon(t)
	commonConfig.SetLocalEnvironment(t)

	bech32Prefix := "wasm"
	gasToken := "cosm"
	params.InitCosmosSdk(bech32Prefix, gasToken)

	clientLogger, err := relaylogger.New()
	require.NoError(t, err, "Could not create relay logger")
	cosmosClient, err := client.NewClient(
		commonConfig.ChainId,
		commonConfig.NodeUrl,
		30*time.Second,
		clientLogger)
	require.NoError(t, err, "Could not create cosmos client")

	nodeName := "primary"
	chainlinkClient, err := common.NewChainlinkClient(commonConfig.Env, commonConfig.ChainId, nodeName, commonConfig.NodeUrl, bech32Prefix, logger)
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

	if useMonitor {
		validateRoundsMonitor(t, *commonConfig, clientLogger, bech32Prefix, gasToken, linkTokenAddress, ocrAddress, ocrProxyAddress)
	} else {
		err = validateRounds(t, cosmosClient, types.MustAccAddressFromBech32(ocrAddress), types.MustAccAddressFromBech32(ocrProxyAddress), commonConfig.IsSoak, commonConfig.TestDuration)
		require.NoError(t, err, "Validating round should not fail")
	}

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

// todo: this currently doesnt support soak testing
func validateRoundsMonitor(t *testing.T, config common.Common, relayLogger relaylogger.Logger, bech32Prefix string, gasToken string, linkTokenAddress string, ocrAddress string, ocrProxyAddress string) {
	logger := common.GetTestLogger(t)

	// used by the chainlink-relay monitoring config parser
	monitorNodesAddress := "127.0.0.1:39010"
	monitorFeedsAddress := "127.0.0.1:39011"
	os.Setenv("NODES_URL", "http://"+monitorNodesAddress)
	os.Setenv("FEEDS_URL", "http://"+monitorFeedsAddress)
	os.Setenv("HTTP_ADDRESS", "localhost:3000")
	os.Setenv("SCHEMA_REGISTRY_URL", "http://localhost:8989")

	os.Setenv("KAFKA_SECURITY_PROTOCOL", "SASL_PLAINTEXT")
	os.Setenv("KAFKA_TRANSMISSION_TOPIC", "transmission_topic")
	os.Setenv("KAFKA_CLIENT_ID", "cosmos")
	os.Setenv("KAFKA_BROKERS", "localhost:29092")
	os.Setenv("KAFKA_CONFIG_SET_SIMPLIFIED_TOPIC", "config_set_simplified")
	os.Setenv("KAFKA_SASL_MECHANISM", "PLAIN")
	os.Setenv("KAFKA_SASL_USERNAME", "user")
	os.Setenv("KAFKA_SASL_PASSWORD", "pass")

	monitorCtx, monitorCancel := context.WithCancel(context.Background())
	monitorConfig := monitoring.CosmosConfig{
		TendermintURL:        config.NodeUrl,
		TendermintReqsPerSec: 2,
		NetworkName:          "primary",
		NetworkID:            "cosmos-network",
		ChainID:              config.ChainId,
		ReadTimeout:          1 * time.Minute,
		PollInterval:         5 * time.Second,
		LinkTokenAddress:     types.MustAccAddressFromBech32(linkTokenAddress),
		Bech32Prefix:         bech32Prefix,
		GasToken:             gasToken,
	}
	monitor, err := monitoring.NewCosmosMonitor(monitorCtx, monitorConfig, relayLogger)
	require.NoError(t, err, "Could not create cosmos monitor")

	nodeConfigJson := fmt.Sprintf("[{\"id\": \"primary\", \"nodeAddress\": [\"%s\"]}]", config.NodeUrl)
	nodeServer := common.RunHTTPServer(t, "monitorNodes", monitorNodesAddress, map[string][]byte{
		"/": []byte(nodeConfigJson),
	})

	feedConfigJson := fmt.Sprintf(`
    [{
      "name": "testfeed",
      "path": "/testpath",
      "symbol": "ABC",
      "contract_address_bech32": "%s",
      "proxy_address_bech32": "%s"
    }]
    `, ocrAddress, ocrProxyAddress)
	feedServer := common.RunHTTPServer(t, "monitorFeeds", monitorFeedsAddress, map[string][]byte{
		"/": []byte(feedConfigJson),
	})

	start := time.Now()
	go func() {
		type TransmissionDetails struct {
			Round        float64
			LatestAnswer float64
		}
		previous := TransmissionDetails{}

		var parser expfmt.TextParser

		// testing variables
		mockAdapterValue := 5 // TODO(BCI-1746): dynamic mock-adapter values
		rounds := 10
		increasing := 0
		stuck := 0

		req, err := http.NewRequest("GET", "http://localhost:3000/metrics", nil)
		require.NoError(t, err, "Could not create request")

		time.Sleep(10 * time.Second) // wait a total of 15 seconds before starting
		// poll prometheus metrics every 5 seconds
		for {
			time.Sleep(5 * time.Second)

			// end condition: enough rounds have occurred
			if increasing >= rounds {
				logger.Info().Msg("Enough rounds have been observed")
				monitorCancel()
				return
			}

			// end condition: rounds have been stuck
			if stuck > 30 {
				require.Fail(t, "rounds have been stuck for too long")
				monitorCancel()
				return
			}

			// end condition: test timeout
			if time.Since(start) > config.TestDuration {
				require.Fail(t, "test timeout")
				monitorCancel()
				return
			}

			// fetch prometheus metrics
			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "Could not get prometheus metrics")
			mf, err := parser.TextToMetricFamilies(res.Body)
			require.NoError(t, err, "Could not parse prometheus metrics")
			res.Body.Close()

			if mf["offchain_aggregator_round_id"] == nil {
				fmt.Println("no transmissions yet")
				continue
			}
			round := mf["offchain_aggregator_round_id"].GetMetric()[0].GetGauge().GetValue()
			fmt.Println("round:", round)
			require.GreaterOrEqual(t, round, previous.Round, "round should be increasing")

			latestAnswer := mf["offchain_aggregator_answers"].GetMetric()[0].GetGauge().GetValue()
			fmt.Println("latest answer:", latestAnswer)
			require.Equal(t, int(latestAnswer), mockAdapterValue, "latest answer should match mock adapter value")

			if round > previous.Round {
				increasing++
				stuck = 0
			} else {
				stuck++
			}

			previous = TransmissionDetails{
				Round:        round,
				LatestAnswer: latestAnswer,
			}
		}
	}()

	logger.Info().Msg("Running monitor...")
	monitor.Run()
	logger.Info().Msg("Monitor stopped.")
	nodeServer.Close()
	feedServer.Close()
}
