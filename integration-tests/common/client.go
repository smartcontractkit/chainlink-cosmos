package common

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-env/environment"
	"github.com/smartcontractkit/chainlink/integration-tests/client"
)

type ChainlinkClient struct {
	ChainlinkNodes []*client.Chainlink
	NodeKeys       []client.NodeKeysBundle
	bTypeAttr      *client.BridgeTypeAttributes
	bootstrapPeers []client.P2PData
}

// CreateKeys Creates node keys and defines chain and nodes for each node
func NewChainlinkClient(env *environment.Environment, chainName string, chainId string, tendermintURL string) (*ChainlinkClient, error) {
	nodes, err := client.ConnectChainlinkNodes(env)
	if err != nil {
		return nil, err
	}
	fmt.Println(nodes)
	nodeKeys, _, err := client.CreateNodeKeysBundle(nodes, chainName, chainId)
	if err != nil {
		return nil, err
	}
	fmt.Println(nodeKeys)
	for _, n := range nodes {
		_, _, err = n.CreateCosmosChain(&client.CosmosChainAttributes{
			ChainID: chainId,
			Config:  client.CosmosChainConfig{},
		})
		if err != nil {
			return nil, err
		}
		_, _, err = n.CreateCosmosNode(&client.CosmosNodeAttributes{
			Name:          chainName,
			CosmosChainID: chainId,
			TendermintURL: tendermintURL,
		})
		if err != nil {
			return nil, err
		}
	}

	return &ChainlinkClient{
		ChainlinkNodes: nodes,
		NodeKeys:       nodeKeys,
	}, nil
}

func (cc *ChainlinkClient) LoadOCR2Config(accountAddresses []string) (*OCR2Config, error) {
	var offChainKeys []string
	var onChainKeys []string
	var peerIds []string
	var txKeys []string
	var cfgKeys []string
	for i, key := range cc.NodeKeys {
		offChainKeys = append(offChainKeys, key.OCR2Key.Data.Attributes.OffChainPublicKey)
		peerIds = append(peerIds, key.PeerID)
		txKeys = append(txKeys, accountAddresses[i])
		onChainKeys = append(onChainKeys, key.OCR2Key.Data.Attributes.OnChainPublicKey)
		cfgKeys = append(cfgKeys, key.OCR2Key.Data.Attributes.ConfigPublicKey)
	}

	var payload = TestOCR2Config
	payload.Signers = onChainKeys
	payload.Transmitters = txKeys
	payload.OffchainConfig.OffchainPublicKeys = offChainKeys
	payload.OffchainConfig.PeerIds = peerIds
	payload.OffchainConfig.ConfigPublicKeys = cfgKeys
	return &payload, nil
}
