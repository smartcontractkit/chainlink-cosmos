package common

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/smartcontractkit/chainlink-env/environment"
	"github.com/smartcontractkit/chainlink-env/pkg/alias"
	"github.com/smartcontractkit/chainlink-env/pkg/helm/chainlink"
	"github.com/smartcontractkit/chainlink-env/pkg/helm/mockserver"
	mockservercfg "github.com/smartcontractkit/chainlink-env/pkg/helm/mockserver-cfg"

	"github.com/smartcontractkit/chainlink-cosmos/ops/wasmd"
)

// TODO: those should be moved as a common part of chainlink-testing-framework

const (
	chainName          = "wasmd"
	chainID            = "testing"
	ChainBlockTime     = "200ms"
	ChainBlockTimeSoak = "2s"
	defaultNodeUrl     = "http://tendermint-rpc:26657"
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
	NodeUrl               string
	Mnemonic              string
	Account               string
	ObservationSource     string
	JuelsPerFeeCoinSource string
	ChainlinkConfig       string
	K8sConfig             *environment.Config
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
		panic(fmt.Sprintf("Please define a proper duration for the test: %v", err))
	}
	t, err := time.ParseDuration(*alias.ShortDur(duration))
	if err != nil {
		panic(fmt.Sprintf("Please define a proper duration for the test: %v", err))
	}
	return t
}

func NewCommon() *Common {
	nodeUrl := getEnv("NODE_URL")
	if nodeUrl == "" {
		nodeUrl = defaultNodeUrl
	}
	chainlinkConfig := fmt.Sprintf(`[[Cosmos]]
Enabled = true
ChainID = '%s'
[[Cosmos.Nodes]]
Name = 'primary'
TendermintURL = '%s'

[OCR2]
Enabled = true

[P2P]
[P2P.V2]
Enabled = true
DeltaDial = '5s'
DeltaReconcile = '5s'
ListenAddresses = ['0.0.0.0:6690']
`, chainID, nodeUrl)
	log.Debug().Str("toml", chainlinkConfig).Msg("Created chainlink config")
	c := &Common{
		IsSoak:                getEnv("SOAK") != "",
		ChainName:             chainName,
		ChainId:               chainID,
		NodeCount:             getNodeCount(),
		TTL:                   getTTL(),
		NodeUrl:               nodeUrl,
		Mnemonic:              getEnv("MNEMONIC"),
		Account:               getEnv("ACCOUNT"),
		ObservationSource:     observationSource,
		JuelsPerFeeCoinSource: juelsPerFeeCoinSource,
		ChainlinkConfig:       chainlinkConfig,
	}
	return c
}

func (c *Common) SetK8sEnvironment(t *testing.T) {
	c.K8sConfig = &environment.Config{
		NamespacePrefix: "cosmos-ocr",
		TTL:             c.TTL,
		Test:            t,
	}
	c.Env = environment.New(c.K8sConfig).
		AddHelm(wasmd.New(nil)).
		AddHelm(mockservercfg.New(nil)).
		AddHelm(mockserver.New(nil)).
		AddHelm(chainlink.New(0, map[string]any{
			"replicas": c.NodeCount,
			"toml":     c.ChainlinkConfig,
		}))
}
