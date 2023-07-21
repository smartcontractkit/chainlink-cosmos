package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	ctfClient "github.com/smartcontractkit/chainlink-testing-framework/client"
	"github.com/smartcontractkit/chainlink/integration-tests/client"

	"github.com/smartcontractkit/chainlink-cosmos/ops/gauntlet"
)

type Test struct {
	// Devnet                *devnet.CosmosDevnetClient
	Cc *ChainlinkClient
	// Cosmos              *cosmos.Client //comes from relayer
	// OCR2Client            *ocr2.Client
	Cg                    *gauntlet.CosmosGauntlet
	mockServer            *ctfClient.MockserverClient
	L1RPCUrl              string
	Common                *Common
	LinkTokenAddr         string
	OCRAddr               string
	AccessControllerAddr  string
	ProxyAddr             string
	ObservationSource     string
	JuelsPerFeeCoinSource string
	T                     *testing.T
}

type ChainlinkClient struct {
	NKeys          []client.NodeKeysBundle
	ChainlinkNodes []*client.Chainlink
	// bTypeAttr      *client.BridgeTypeAttributes
	// bootstrapPeers []client.P2PData
}

// DeployCluster Deploys and sets up config of the environment and nodes
func (testState *Test) DeployCluster() {
	// lggr := logger.Nop()
	testState.Cc = &ChainlinkClient{}
	testState.ObservationSource = testState.GetDefaultObservationSource()
	testState.JuelsPerFeeCoinSource = testState.GetDefaultJuelsPerFeeCoinSource()
	testState.DeployEnv()
	if testState.Common.Env.WillUseRemoteRunner() {
		return // short circuit here if using a remote runner
	}
	// from starknet, may be useful later
	// testState.SetupClients()
	// if testState.Common.Testnet {
	// 	testState.Common.Env.URLs[testState.Common.ServiceKeyL2][1] = testState.Common.L2RPCUrl
	// }
	// var err error
	// testState.Cc.NKeys, testState.Cc.ChainlinkNodes, err = testState.Common.CreateKeys(testState.Common.Env)
	// require.NoError(testState.T, err, "Creating chains and keys should not fail")
	// testState.Cosmos, err = starknet.NewClient(testState.Common.ChainId, testState.Common.L2RPCUrl, lggr, &rpcRequestTimeout)
	// require.NoError(testState.T, err, "Creating starknet client should not fail")
	// testState.OCR2Client, err = ocr2.NewClient(testState.Starknet, lggr)
	// require.NoError(testState.T, err, "Creating ocr2 client should not fail")
	// if !testState.Common.Testnet {
	// 	err = os.Setenv("PRIVATE_KEY", testState.GetDefaultPrivateKey())
	// 	require.NoError(testState.T, err, "Setting private key should not fail")
	// 	err = os.Setenv("ACCOUNT", testState.GetDefaultWalletAddress())
	// 	require.NoError(testState.T, err, "Setting account address should not fail")
	// 	testState.Devnet.AutoDumpState() // Auto dumping devnet state to avoid losing contracts on crash
	// }
}

// DeployEnv Deploys the environment
func (testState *Test) DeployEnv() {
	err := testState.Common.Env.Run()
	require.NoError(testState.T, err)
	if testState.Common.Env.WillUseRemoteRunner() {
		return // short circuit here if using a remote runner
	}
	testState.mockServer, err = ctfClient.ConnectMockServer(testState.Common.Env)
	require.NoError(testState.T, err, "Creating mockserver clients shouldn't fail")
}

func (testState *Test) GetDefaultObservationSource() string {
	return `
			val [type = "bridge" name="bridge-mockserver"]
			parse [type="jsonparse" path="data,result"]
			val -> parse
			`
}

func (testState *Test) GetDefaultJuelsPerFeeCoinSource() string {
	return `"""
			sum  [type="sum" values=<[451000]> ]
			sum
			"""
			`
}
