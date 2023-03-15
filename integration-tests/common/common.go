package common

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/smartcontractkit/chainlink-cosmos/ops/wasmd"
	"github.com/smartcontractkit/chainlink-env/environment"
	"github.com/smartcontractkit/chainlink-env/pkg/alias"
	"github.com/smartcontractkit/chainlink-env/pkg/helm/chainlink"
	"github.com/smartcontractkit/chainlink-env/pkg/helm/mockserver"
	mockservercfg "github.com/smartcontractkit/chainlink-env/pkg/helm/mockserver-cfg"
)

// TODO: those should be moved as a common part of chainlink-testing-framework

const (
	chainName          = "wasmd"
	chainID            = "testing"
	ChainBlockTime     = "200ms"
	ChainBlockTimeSoak = "2s"
)

type Common struct {
	P2PPort    string
	ChainName  string
	ChainId    string
	NodeCount  int
	TTL        time.Duration
	Testnet    bool
	L2RPCUrl   string
	PrivateKey string
	Account    string
	ClConfig   map[string]interface{}
	K8Config   *environment.Config
	Env        *environment.Environment
}

// getEnv gets the environment variable if it exists and sets it for the remote runner
func getEnv(v string) string {
	val := os.Getenv(v)
	if val != "" {
		os.Setenv(fmt.Sprintf("TEST_%s", v), val)
	}
	return val
}

func New() *Common {
	var err error
	c := &Common{
		ChainName: chainName,
		ChainId:   chainID,
	}
	// Checking if count of OCR nodes is defined in ENV
	nodeCountSet := getEnv("NODE_COUNT")
	if nodeCountSet != "" {
		c.NodeCount, err = strconv.Atoi(nodeCountSet)
		if err != nil {
			panic(fmt.Sprintf("Please define a proper node count for the test: %v", err))
		}
	} else {
		panic("Please define NODE_COUNT")
	}

	// Checking if TTL env var is set in ENV
	ttlValue := getEnv("TTL")
	if ttlValue != "" {
		duration, err := time.ParseDuration(ttlValue)
		if err != nil {
			panic(fmt.Sprintf("Please define a proper duration for the test: %v", err))
		}
		c.TTL, err = time.ParseDuration(*alias.ShortDur(duration))
		if err != nil {
			panic(fmt.Sprintf("Please define a proper duration for the test: %v", err))
		}
	} else {
		panic("Please define TTL of env")
	}

	// Setting optional parameters
	c.L2RPCUrl = getEnv("L2_RPC_URL") // Fetch L2 RPC url if defined
	c.Testnet = c.L2RPCUrl != ""
	c.PrivateKey = getEnv("PRIVATE_KEY")
	c.Account = getEnv("ACCOUNT")

	return c
}

func (c *Common) Default(t *testing.T) {
	c.K8Config = &environment.Config{NamespacePrefix: "chainlink-ocr-cosmos", TTL: c.TTL, Test: t}
	wasmdUrl := fmt.Sprintf("http://%s:%d", chainName, 9091)
	if c.Testnet {
		wasmdUrl = c.L2RPCUrl
	}
	baseTOML := fmt.Sprintf(`[[Cosmos]]
Enabled = true
ChainID = '%s'

[[Cosmos.Nodes]]
Name = 'primary'
URL = '%s'

[OCR2]
Enabled = true

[P2P]
[P2P.V2]
Enabled = true
DeltaDial = '5s'
DeltaReconcile = '5s'
ListenAddresses = ['0.0.0.0:6690']
`, c.ChainId, wasmdUrl)
	log.Debug().Str("toml", baseTOML).Msg("TOML")
	c.ClConfig = map[string]interface{}{
		"replicas": c.NodeCount,
		"toml":     baseTOML,
	}
	c.Env = environment.New(c.K8Config).
		AddHelm(wasmd.New(nil, "")).
		AddHelm(mockservercfg.New(nil)).
		AddHelm(mockserver.New(nil)).
		AddHelm(chainlink.New(0, c.ClConfig))
}
