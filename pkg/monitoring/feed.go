package monitoring

import (
	"encoding/json"
	"fmt"
	"io"

	sdk "github.com/cosmos/cosmos-sdk/types"
	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
)

type TerraFeedConfig struct {
	Name           string `json:"name,omitempty"`
	Path           string `json:"path,omitempty"`
	Symbol         string `json:"symbol,omitempty"`
	HeartbeatSec   int64  `json:"heartbeat,omitempty"`
	ContractType   string `json:"contract_type,omitempty"`
	ContractStatus string `json:"status,omitempty"`

	ContractAddressBech32 string         `json:"contract_address_bech32,omitempty"`
	ContractAddress       sdk.AccAddress `json:"-"`
}

var _ relayMonitoring.FeedConfig = TerraFeedConfig{}

func (t TerraFeedConfig) GetName() string {
	return t.Name
}

func (t TerraFeedConfig) GetPath() string {
	return t.Path
}

func (t TerraFeedConfig) GetFeedPath() string {
	return t.Path
}

func (t TerraFeedConfig) GetSymbol() string {
	return t.Symbol
}

func (t TerraFeedConfig) GetHeartbeatSec() int64 {
	return t.HeartbeatSec
}

func (t TerraFeedConfig) GetContractType() string {
	return t.ContractType
}

func (t TerraFeedConfig) GetContractStatus() string {
	return t.ContractStatus
}

func (t TerraFeedConfig) GetContractAddress() string {
	return t.ContractAddressBech32
}

func (t TerraFeedConfig) GetContractAddressBytes() []byte {
	return t.ContractAddress.Bytes()
}

func (t TerraFeedConfig) ToMapping() map[string]interface{} {
	return map[string]interface{}{
		"feed_name":        t.Name,
		"feed_path":        t.Path,
		"symbol":           t.Symbol,
		"heartbeat_sec":    int64(t.HeartbeatSec),
		"contract_type":    t.ContractType,
		"contract_status":  t.ContractStatus,
		"contract_address": t.ContractAddress.Bytes(),
	}
}

func TerraFeedParser(buf io.ReadCloser) ([]relayMonitoring.FeedConfig, error) {
	rawFeeds := []TerraFeedConfig{}
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
		feeds[i] = relayMonitoring.FeedConfig(TerraFeedConfig{
			rawFeed.Name,
			rawFeed.Path,
			rawFeed.Symbol,
			rawFeed.HeartbeatSec,
			rawFeed.ContractType,
			rawFeed.ContractStatus,

			rawFeed.ContractAddressBech32,
			contractAddress,
		})
	}
	return feeds, nil
}
