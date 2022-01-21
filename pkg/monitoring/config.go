package monitoring

import (
	"fmt"
	"net/url"
	"os"
	"time"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
)

type TerraConfig struct {
	TendermintURL string
	FCDURL        string
	NetworkName   string
	NetworkID     string
	ChainID       string
	ReadTimeout   time.Duration
	PollInterval  time.Duration
}

var _ relayMonitoring.ChainConfig = TerraConfig{}

func (t TerraConfig) GetRPCEndpoint() string         { return t.TendermintURL }
func (t TerraConfig) GetNetworkName() string         { return t.NetworkName }
func (t TerraConfig) GetNetworkID() string           { return t.NetworkID }
func (t TerraConfig) GetChainID() string             { return t.ChainID }
func (t TerraConfig) GetReadTimeout() time.Duration  { return t.ReadTimeout }
func (t TerraConfig) GetPollInterval() time.Duration { return t.PollInterval }

func (t TerraConfig) ToMapping() map[string]interface{} {
	return map[string]interface{}{
		"network_name": t.NetworkName,
		"network_id":   t.NetworkID,
		"chain_id":     t.ChainID,
	}
}

func ParseTerraConfig() (TerraConfig, error) {
	cfg := TerraConfig{}

	if err := parseEnvVars(&cfg); err != nil {
		return cfg, err
	}

	applyDefaults(&cfg)

	err := validateConfig(cfg)
	return cfg, err
}

func parseEnvVars(cfg *TerraConfig) error {
	if value, isPresent := os.LookupEnv("TERRA_TENDERMINT_URL"); isPresent {
		cfg.TendermintURL = value
	}
	if value, isPresent := os.LookupEnv("TERRA_FCD_URL"); isPresent {
		cfg.FCDURL = value
	}
	if value, isPresent := os.LookupEnv("TERRA_NETWORK_NAME"); isPresent {
		cfg.NetworkName = value
	}
	if value, isPresent := os.LookupEnv("TERRA_NETWORK_ID"); isPresent {
		cfg.NetworkID = value
	}
	if value, isPresent := os.LookupEnv("TERRA_CHAIN_ID"); isPresent {
		cfg.ChainID = value
	}
	if value, isPresent := os.LookupEnv("TERRA_READ_TIMEOUT"); isPresent {
		readTimeout, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("failed to parse env var TERRA_READ_TIMEOUT, see https://pkg.go.dev/time#ParseDuration: %w", err)
		}
		cfg.ReadTimeout = readTimeout
	}
	if value, isPresent := os.LookupEnv("TERRA_POLL_INTERVAL"); isPresent {
		pollInterval, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("failed to parse env var TERRA_POLL_INTERVAL, see https://pkg.go.dev/time#ParseDuration: %w", err)
		}
		cfg.PollInterval = pollInterval
	}
	return nil
}

func validateConfig(cfg TerraConfig) error {
	// Required config
	for envVarName, currentValue := range map[string]string{
		"TERRA_TENDERMINT_URL": cfg.TendermintURL,
		"TERRA_FCD_URL":        cfg.FCDURL,
		"TERRA_NETWORK_NAME":   cfg.NetworkName,
		"TERRA_NETWORK_ID":     cfg.NetworkID,
		"TERRA_CHAIN_ID":       cfg.ChainID,
	} {
		if currentValue == "" {
			return fmt.Errorf("'%s' env var is required", envVarName)
		}
	}
	// Validate URLs.
	for envVarName, currentValue := range map[string]string{
		"TERRA_TENDERMINT_URL": cfg.TendermintURL,
		"TERRA_FCD_URL":        cfg.FCDURL,
	} {
		if _, err := url.ParseRequestURI(currentValue); err != nil {
			return fmt.Errorf("%s='%s' is not a valid URL: %w", envVarName, currentValue, err)
		}
	}
	return nil
}

func applyDefaults(cfg *TerraConfig) {
	if cfg.FCDURL == "" {
		cfg.FCDURL = "https://fcd.terra.dev"
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 2 * time.Second
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 5 * time.Second
	}
}
