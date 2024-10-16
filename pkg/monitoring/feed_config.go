package monitoring

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	relayMonitoring "github.com/smartcontractkit/chainlink-common/pkg/monitoring"
)

// CosmosFeedConfig holds data extracted from the RDD
type CosmosFeedConfig struct {
	Name           string   `json:"name,omitempty"`
	Path           string   `json:"path,omitempty"`
	Symbol         string   `json:"symbol,omitempty"`
	HeartbeatSec   int64    `json:"heartbeat,omitempty"`
	ContractType   string   `json:"contract_type,omitempty"`
	ContractStatus string   `json:"status,omitempty"`
	MultiplyRaw    string   `json:"multiply,omitempty"`
	Multiply       *big.Int `json:"-"`

	ContractAddressBech32 string         `json:"contract_address_bech32,omitempty"`
	ContractAddress       sdk.AccAddress `json:"-"`

	// Optional fields! Internal feeds are not proxied. Check ProxyAddressBech32 == ""!
	ProxyAddressBech32 string         `json:"proxy_address_bech32,omitempty"`
	ProxyAddress       sdk.AccAddress `json:"-"`
}

var _ relayMonitoring.FeedConfig = CosmosFeedConfig{}

// GetID returns the feed's contract address encoded as Bech32 which is the feed identifier on Cosmos.
func (t CosmosFeedConfig) GetID() string {
	return t.ContractAddressBech32
}

// GetName returns the feed's name.
func (t CosmosFeedConfig) GetName() string {
	return t.Name
}

// GetPath returns the feed's path.
func (t CosmosFeedConfig) GetPath() string {
	return t.Path
}

// GetSymbol returns the feed's symbol.
func (t CosmosFeedConfig) GetSymbol() string {
	return t.Symbol
}

// GetHeartbeatSec returns the feed's heartbeat in seconds.
func (t CosmosFeedConfig) GetHeartbeatSec() int64 {
	return t.HeartbeatSec
}

// GetContractType returns the feed's contract type.
func (t CosmosFeedConfig) GetContractType() string {
	return t.ContractType
}

// GetContractStatus returns the feed's contract status.
func (t CosmosFeedConfig) GetContractStatus() string {
	return t.ContractStatus
}

// GetContractAddress returns the feed's contract address encoded as Bech32.
func (t CosmosFeedConfig) GetContractAddress() string {
	return t.ContractAddressBech32
}

// GetContractAddressBytes returns the feed's contract address in raw bytes.
func (t CosmosFeedConfig) GetContractAddressBytes() []byte {
	return t.ContractAddress.Bytes()
}

// GetMultiply returns the feed's multiplication factor for updates.
func (t CosmosFeedConfig) GetMultiply() *big.Int {
	return t.Multiply
}

// ToMapping returns the feed's configuration mapped to a data structure expected by the Avro schema encoders.
func (t CosmosFeedConfig) ToMapping() map[string]interface{} {
	return map[string]interface{}{
		"feed_name":               t.Name,
		"feed_path":               t.Path,
		"symbol":                  t.Symbol,
		"heartbeat_sec":           t.HeartbeatSec,
		"contract_type":           t.ContractType,
		"contract_status":         t.ContractStatus,
		"contract_address":        t.ContractAddress.Bytes(),
		"contract_address_string": map[string]interface{}{"string": t.ContractAddressBech32},

		// These fields are legacy. They are required in the schema but they
		// should be set to a zero value for any other chain.
		"transmissions_account": []byte{},
		"state_account":         []byte{},
	}
}

// CosmosFeedsParser decodes a JSON-encoded list of cosmos-specific feed configurations.
func CosmosFeedsParser(buf io.ReadCloser) ([]relayMonitoring.FeedConfig, error) {
	rawFeeds := []CosmosFeedConfig{}
	decoder := json.NewDecoder(buf)
	if err := decoder.Decode(&rawFeeds); err != nil {
		return nil, fmt.Errorf("unable to unmarshal feeds config data: %w", err)
	}
	feeds := make([]relayMonitoring.FeedConfig, len(rawFeeds))
	for i, rawFeed := range rawFeeds {
		contractAddress, err := sdk.AccAddressFromBech32(rawFeed.ContractAddressBech32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse contract address '%s' from JSON at index i=%d: %w", rawFeed.ContractAddressBech32, i, err)
		}
		var proxyAddress sdk.AccAddress
		if rawFeed.ProxyAddressBech32 != "" {
			address, err := sdk.AccAddressFromBech32(rawFeed.ProxyAddressBech32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse proxy contract address '%s' from JSON at index i=%d: %w", rawFeed.ProxyAddressBech32, i, err)
			}
			proxyAddress = address
		}
		multiply, ok := new(big.Int).SetString(rawFeed.MultiplyRaw, 10)
		if !ok {
			return nil, fmt.Errorf("failed to parse multiply '%s' into a big.Int", rawFeed.MultiplyRaw)
		}
		// NOTE: multiply is not required so if a parse error occurs, we'll use 0.
		feeds[i] = relayMonitoring.FeedConfig(CosmosFeedConfig{
			rawFeed.Name,
			rawFeed.Path,
			rawFeed.Symbol,
			rawFeed.HeartbeatSec,
			rawFeed.ContractType,
			rawFeed.ContractStatus,
			rawFeed.MultiplyRaw,
			multiply,
			rawFeed.ContractAddressBech32,
			contractAddress,
			rawFeed.ProxyAddressBech32,
			proxyAddress,
		})
	}
	return feeds, nil
}
