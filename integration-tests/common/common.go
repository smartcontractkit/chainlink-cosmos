package common

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-env/environment"
	"github.com/smartcontractkit/chainlink-env/pkg/alias"
	"github.com/smartcontractkit/chainlink-env/pkg/helm/chainlink"
	"github.com/smartcontractkit/chainlink-env/pkg/helm/mockserver"
	mockservercfg "github.com/smartcontractkit/chainlink-env/pkg/helm/mockserver-cfg"

	"github.com/smartcontractkit/chainlink-cosmos/ops/wasmd"
)

// TODO: those should be moved as a common part of chainlink-testing-framework

const (
	chainName              = "cosmos"
	chainID                = "testing"
	ChainBlockTime         = "200ms"
	ChainBlockTimeSoak     = "2s"
	defaultNodeUrl         = "http://127.0.0.1:26657"
	defaultInternalNodeUrl = "http://host.docker.internal:26657"
)

var (
	observationSource = `
			val [type="bridge" name="bridge-coinmetrics" requestData=<{"data": {"from":"LINK","to":"USD"}}>]
			parse [type="jsonparse" path="result"]
			val -> parse
			`
	juelsPerFeeCoinSource = `"""
			sum  [type="sum" values=<[451000]> ]
			sum
			"""
			`
)

type Common struct {
	IsSoak                bool
	P2PPort               string
	ChainName             string
	ChainId               string
	NodeCount             int
	TTL                   time.Duration
	TestDuration          time.Duration
	NodeUrl               string
	MockUrl               string
	Mnemonic              string
	ObservationSource     string
	JuelsPerFeeCoinSource string
	ChainlinkConfig       string
	Env                   *environment.Environment
}

// getEnv gets the environment variable if it exists and sets it for the remote runner
func getEnv(v string) string {
	val := os.Getenv(v)
	if val != "" {
		os.Setenv(fmt.Sprintf("TEST_%s", v), val)
	}
	return val
}

func getNodeCount() int {
	// Checking if count of OCR nodes is defined in ENV
	nodeCountSet := getEnv("NODE_COUNT")
	if nodeCountSet == "" {
		panic("Please define NODE_COUNT")
	}
	nodeCount, err := strconv.Atoi(nodeCountSet)
	if err != nil {
		panic(fmt.Sprintf("Please define a proper node count for the test: %v", err))
	}
	return nodeCount
}

func getTTL() time.Duration {
	ttlValue := getEnv("TTL")
	if ttlValue == "" {
		panic("Please define TTL of env")
	}
	duration, err := time.ParseDuration(ttlValue)
	if err != nil {
		panic(fmt.Sprintf("Please define a proper TTL for the test: %v", err))
	}
	t, err := time.ParseDuration(*alias.ShortDur(duration))
	if err != nil {
		panic(fmt.Sprintf("Please define a proper TTL for the test: %v", err))
	}
	return t
}

func getTestDuration() time.Duration {
	testDurationValue := getEnv("TEST_DURATION")
	if testDurationValue == "" {
		return time.Duration(time.Minute * 15)
	}
	duration, err := time.ParseDuration(testDurationValue)
	if err != nil {
		panic(fmt.Sprintf("Please define a proper duration for the test: %v", err))
	}
	t, err := time.ParseDuration(*alias.ShortDur(duration))
	if err != nil {
		panic(fmt.Sprintf("Please define a proper duration for the test: %v", err))
	}
	return t
}

func NewCommon(t *testing.T) *Common {
	nodeUrl := getEnv("NODE_URL")
	if nodeUrl == "" {
		nodeUrl = defaultNodeUrl
	}
	internalNodeUrl := getEnv("INTERNAL_NODE_URL")
	if internalNodeUrl == "" {
		internalNodeUrl = defaultInternalNodeUrl
	}
	chainlinkConfig := fmt.Sprintf(`[[Cosmos]]
Enabled = true
ChainID = '%s'
Bech32Prefix = 'wasm'
FeeToken = 'ucosm'

[[Cosmos.Nodes]]
Name = 'primary'
TendermintURL = '%s'

[OCR2]
Enabled = true

[P2P]
[P2P.V1]
Enabled = false
[P2P.V2]
Enabled = true
DeltaDial = '5s'
DeltaReconcile = '5s'
ListenAddresses = ['0.0.0.0:6691']

[WebServer]
HTTPPort = 6688
[WebServer.TLS]
HTTPSPort = 0
`, chainID, internalNodeUrl)
	log.Debug().Str("toml", chainlinkConfig).Msg("Created chainlink config")

	ttl := getTTL()

	envConfig := &environment.Config{
		NamespacePrefix: "cosmos-ocr",
		TTL:             ttl,
		Test:            t,
	}
	c := &Common{
		IsSoak:                getEnv("SOAK") != "",
		ChainName:             chainName,
		ChainId:               chainID,
		NodeCount:             getNodeCount(),
		TTL:                   getTTL(),
		TestDuration:          getTestDuration(),
		NodeUrl:               nodeUrl,
		MockUrl:               "http://172.17.0.1:6060",
		Mnemonic:              getEnv("MNEMONIC"),
		ObservationSource:     observationSource,
		JuelsPerFeeCoinSource: juelsPerFeeCoinSource,
		ChainlinkConfig:       chainlinkConfig,
		Env:                   environment.New(envConfig),
	}
	return c
}

func (c *Common) SetLocalEnvironment(t *testing.T) {
	// Run scripts to set up local test environment
	log.Info().Msg("Starting wasmd container...")
	err := exec.Command("../scripts/wasmd.sh").Run()
	require.NoError(t, err, "Could not start wasmd container")
	log.Info().Msg("Starting postgres container...")
	err = exec.Command("../scripts/postgres.sh").Run()
	require.NoError(t, err, "Could not start postgres container")
	log.Info().Msg("Starting mock adapter...")
	err = exec.Command("../scripts/mock-adapter.sh").Run()
	require.NoError(t, err, "Could not start mock adapter")
	log.Info().Msg("Starting core nodes...")
	cmd := exec.Command("../scripts/core.sh")
	cmd.Env = append(os.Environ(), fmt.Sprintf("CL_CONFIG=%s", c.ChainlinkConfig))
	err = cmd.Run()
	require.NoError(t, err, "Could not start core nodes")
	log.Info().Msg("Set up local stack complete.")

	// Set ChainlinkNodeDetails
	var nodeDetails []*environment.ChainlinkNodeDetail
	var basePort = 50100
	for i := 0; i < c.NodeCount; i++ {
		dbLocalIP := fmt.Sprintf("postgresql://postgres:postgres@host.docker.internal:5432/cosmos_test_%d?sslmode=disable", i+1)
		nodeDetails = append(nodeDetails, &environment.ChainlinkNodeDetail{
			ChartName:  "unused",
			PodName:    "unused",
			LocalIP:    "http://127.0.0.1:" + strconv.Itoa(basePort+i),
			InternalIP: "http://host.docker.internal:" + strconv.Itoa(basePort+i),
			DBLocalIP:  dbLocalIP,
		})
	}
	c.Env.ChainlinkNodeDetails = nodeDetails
}

func (c *Common) TearDownLocalEnvironment(t *testing.T) {
	log.Info().Msg("Tearing down core nodes...")
	err := exec.Command("../scripts/core.down.sh").Run()
	require.NoError(t, err, "Could not tear down core nodes")
	log.Info().Msg("Tearing down mock adapter...")
	err = exec.Command("../scripts/mock-adapter.down.sh").Run()
	require.NoError(t, err, "Could not tear down mock adapter")
	log.Info().Msg("Tearing down postgres container...")
	err = exec.Command("../scripts/postgres.down.sh").Run()
	require.NoError(t, err, "Could not tear down postgres container")
	log.Info().Msg("Tearing down wasmd container...")
	err = exec.Command("../scripts/wasmd.down.sh").Run()
	require.NoError(t, err, "Could not tear down wasmd container")
	log.Info().Msg("Tear down local stack complete.")
}

func (c *Common) SetK8sEnvironment() {
	c.Env.AddHelm(wasmd.New(nil)).
		AddHelm(mockservercfg.New(nil)).
		AddHelm(mockserver.New(nil)).
		AddHelm(chainlink.New(0, map[string]any{
			"replicas": c.NodeCount,
			"toml":     c.ChainlinkConfig,
		}))
}
