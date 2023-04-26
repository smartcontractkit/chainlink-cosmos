package monitoring

import (
	"encoding/json"
	"fmt"
	"io"

	relayMonitoring "github.com/smartcontractkit/chainlink-relay/pkg/monitoring"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

type CosmosNodeConfig struct {
	ID          string   `json:"id,omitempty"`
	NodeAddress []string `json:"nodeAddress,omitempty"`
}

func (t CosmosNodeConfig) GetName() string {
	return t.ID
}

func (t CosmosNodeConfig) GetAccount() types.Account {
	address := ""
	if len(t.NodeAddress) != 0 {
		address = t.NodeAddress[0]
	}
	return types.Account(address)
}

func CosmosNodesParser(buf io.ReadCloser) ([]relayMonitoring.NodeConfig, error) {
	rawNodes := []CosmosNodeConfig{}
	decoder := json.NewDecoder(buf)
	if err := decoder.Decode(&rawNodes); err != nil {
		return nil, fmt.Errorf("unable to unmarshal nodes config data: %w", err)
	}
	nodes := make([]relayMonitoring.NodeConfig, len(rawNodes))
	for i, rawNode := range rawNodes {
		nodes[i] = rawNode
	}
	return nodes, nil
}
