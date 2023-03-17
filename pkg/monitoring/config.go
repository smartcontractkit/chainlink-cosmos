package monitoring

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
)

// CosmosConfig contains configuration for connecting to a cosmos RPC client.
type CosmosConfig struct {
	TendermintURL        string
	TendermintReqsPerSec int
	LCDURL               string
	LCDReqsPerSec        int
	NetworkName          string
	NetworkID            string
	ChainID              string
	ReadTimeout          time.Duration
	PollInterval         time.Duration
	LinkTokenAddress     sdk.AccAddress
}

var _ relayMonitoring.ChainConfig = CosmosConfig{}

// GetRPCEndpoint return the tendermint url of a Cosmos client.
func (t CosmosConfig) GetRPCEndpoint() string { return t.TendermintURL }

// GetNetworkName returns the network name.
func (t CosmosConfig) GetNetworkName() string { return t.NetworkName }

// GetNetworkID returns the network id.
func (t CosmosConfig) GetNetworkID() string { return t.NetworkID }

// GetChainID returns the chain id.
func (t CosmosConfig) GetChainID() string { return t.ChainID }

// GetReadTimeout returns the max allowed duration of a request to a Cosmos client.
func (t CosmosConfig) GetReadTimeout() time.Duration { return t.ReadTimeout }

// GetPollInterval returns the interval at which data from the chain is read.
func (t CosmosConfig) GetPollInterval() time.Duration { return t.PollInterval }

// ToMapping returns a data structure expected by the Avro schema encoders.
func (t CosmosConfig) ToMapping() map[string]interface{} {
	return map[string]interface{}{
		"network_name": t.NetworkName,
		"network_id":   t.NetworkID,
		"chain_id":     t.ChainID,
	}
}

// ParseCosmosConfig extracts chain specific configuration from env vars.
func ParseCosmosConfig() (CosmosConfig, error) {
	cfg := CosmosConfig{}

	if err := parseEnvVars(&cfg); err != nil {
		return cfg, err
	}

	applyDefaults(&cfg)

	err := validateConfig(cfg)
	return cfg, err
}

func parseEnvVars(cfg *CosmosConfig) error {
	if value, isPresent := os.LookupEnv("COSMOS_TENDERMINT_URL"); isPresent {
		cfg.TendermintURL = value
	}
	if value, isPresent := os.LookupEnv("COSMOS_TENDERMINT_REQS_PER_SEC"); isPresent {
		rps, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("failed to parse env var COSMOS_TENDERMINT_REQS_PER_SEC '%s' as int: %w", value, err)
		}
		cfg.TendermintReqsPerSec = rps
	}
	if value, isPresent := os.LookupEnv("COSMOS_LCD_URL"); isPresent {
		cfg.LCDURL = value
	}
	if value, isPresent := os.LookupEnv("COSMOS_LCD_REQS_PER_SEC"); isPresent {
		rps, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("failed to parse env var COSMOS_LCD_REQS_PER_SEC '%s' as int: %w", value, err)
		}
		cfg.LCDReqsPerSec = rps
	}
	if value, isPresent := os.LookupEnv("COSMOS_NETWORK_NAME"); isPresent {
		cfg.NetworkName = value
	}
	if value, isPresent := os.LookupEnv("COSMOS_NETWORK_ID"); isPresent {
		cfg.NetworkID = value
	}
	if value, isPresent := os.LookupEnv("COSMOS_CHAIN_ID"); isPresent {
		cfg.ChainID = value
	}
	if value, isPresent := os.LookupEnv("COSMOS_READ_TIMEOUT"); isPresent {
		readTimeout, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("failed to parse env var COSMOS_READ_TIMEOUT, see https://pkg.go.dev/time#ParseDuration: %w", err)
		}
		cfg.ReadTimeout = readTimeout
	}
	if value, isPresent := os.LookupEnv("COSMOS_POLL_INTERVAL"); isPresent {
		pollInterval, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("failed to parse env var COSMOS_POLL_INTERVAL, see https://pkg.go.dev/time#ParseDuration: %w", err)
		}
		cfg.PollInterval = pollInterval
	}
	if value, isPresent := os.LookupEnv("COSMOS_LINK_TOKEN_ADDRESS"); isPresent {
		address, err := sdk.AccAddressFromBech32(value)
		if err != nil {
			return fmt.Errorf("failed to parse the bech32-encoded link token address from '%s': %w", value, err)
		}
		cfg.LinkTokenAddress = address
	}
	return nil
}

func validateConfig(cfg CosmosConfig) error {
	// Required config
	for envVarName, currentValue := range map[string]string{
		"COSMOS_TENDERMINT_URL": cfg.TendermintURL,
		"COSMOS_LCD_URL":        cfg.LCDURL,
		"COSMOS_NETWORK_NAME":   cfg.NetworkName,
		"COSMOS_NETWORK_ID":     cfg.NetworkID,
		"COSMOS_CHAIN_ID":       cfg.ChainID,
	} {
		if currentValue == "" {
			return fmt.Errorf("'%s' env var is required", envVarName)
		}
	}
	// Validate URLs.
	for envVarName, currentValue := range map[string]string{
		"COSMOS_TENDERMINT_URL": cfg.TendermintURL,
		"COSMOS_LCD_URL":        cfg.LCDURL,
	} {
		if _, err := url.ParseRequestURI(currentValue); currentValue != "" && err != nil {
			return fmt.Errorf("%s='%s' is not a valid URL: %w", envVarName, currentValue, err)
		}
	}
	return nil
}

func applyDefaults(cfg *CosmosConfig) {
	if cfg.TendermintReqsPerSec == 0 {
		cfg.TendermintReqsPerSec = 1
	}
	if cfg.LCDURL == "" {
		cfg.LCDURL = "https://lcd.tender.mint/"
	}
	if cfg.LCDReqsPerSec == 0 {
		cfg.LCDReqsPerSec = 1
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 2 * time.Second
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 5 * time.Second
	}
}
