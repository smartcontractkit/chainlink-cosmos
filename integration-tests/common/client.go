package common

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gopkg.in/guregu/null.v4"

	"github.com/smartcontractkit/chainlink-env/environment"
	"github.com/smartcontractkit/chainlink/integration-tests/client"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/relay"
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
	if nodes == nil || len(nodes) == 0 {
		return nil, errors.New("No connected nodes")
	}
	nodeKeys, _, err := client.CreateNodeKeysBundle(nodes, chainName, chainId)
	if err != nil {
		return nil, err
	}
	if nodeKeys == nil || len(nodeKeys) == 0 {
		return nil, errors.New("No node keys")
	}
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

func (cc *ChainlinkClient) GetNodeAddresses() []string {
	var addresses []string
	for _, nodeKey := range cc.NodeKeys {
		addresses = append(addresses, nodeKey.TXKey.Data.Attributes.PublicKey)
	}
	return addresses
}

func (cc *ChainlinkClient) LoadOCR2Config(proposalId string) (*OCR2Config, error) {
	var offChainKeys []string
	var onChainKeys []string
	var peerIds []string
	var txKeys []string
	var cfgKeys []string
	for i, key := range cc.NodeKeys {
		offChainKeys = append(offChainKeys, key.OCR2Key.Data.Attributes.OffChainPublicKey)
		peerIds = append(peerIds, key.PeerID)
		// TODO: This uses a hardcoded array of test addresses with 'wasm' bech32 prefix as the keystore generates
		// addresses with the 'cosmos' prefix by default. We can use  key.TXKey.Data.ID after refactoring
		// the keystore to allow bech32 prefixes to be defined.
		txKeys = append(txKeys, TestTxKeys[i])
		// txKeys = append(txKeys, key.TXKey.Data.ID)
		onChainKeys = append(onChainKeys, key.OCR2Key.Data.Attributes.OnChainPublicKey)
		cfgKeys = append(cfgKeys, key.OCR2Key.Data.Attributes.ConfigPublicKey)
	}
	var payload = TestOCR2Config
	payload.ProposalId = proposalId
	payload.Signers = onChainKeys
	payload.Transmitters = txKeys
	payload.OffchainConfig.OffchainPublicKeys = offChainKeys
	payload.OffchainConfig.PeerIds = peerIds
	payload.OffchainConfig.ConfigPublicKeys = cfgKeys
	return &payload, nil
}

// CreateJobsForContract Creates and sets up the boostrap jobs as well as OCR jobs
func (cc *ChainlinkClient) CreateJobsForContract(chainId, p2pPort, mockUrl string, observationSource string, juelsPerFeeCoinSource string, ocrControllerAddress string) error {
	// TODO: fix up relay configs
	// Define node[0] as bootstrap node
	cc.bootstrapPeers = []client.P2PData{
		{
			InternalIP:   cc.ChainlinkNodes[0].InternalIP(),
			InternalPort: p2pPort,
			PeerID:       cc.NodeKeys[0].PeerID,
		},
	}

	// Defining relay config
	bootstrapRelayConfig := job.JSONConfig{
		"nodeName": fmt.Sprintf("\"cosmos-OCRv2-%s-%s\"", "node", uuid.NewString()),
		"chainID":  fmt.Sprintf("\"%s\"", chainId),
	}

	oracleSpec := job.OCR2OracleSpec{
		ContractID:                  ocrControllerAddress,
		Relay:                       relay.Cosmos,
		RelayConfig:                 bootstrapRelayConfig,
		ContractConfigConfirmations: 1, // don't wait for confirmation on devnet
	}
	// Setting up bootstrap node
	jobSpec := &client.OCR2TaskJobSpec{
		Name:           fmt.Sprintf("cosmos-OCRv2-%s-%s", "bootstrap", uuid.NewString()),
		JobType:        "bootstrap",
		OCR2OracleSpec: oracleSpec,
	}

	_, _, err := cc.ChainlinkNodes[0].CreateJob(jobSpec)
	if err != nil {
		return err
	}

	var p2pBootstrappers []string

	for i := range cc.bootstrapPeers {
		p2pBootstrappers = append(p2pBootstrappers, cc.bootstrapPeers[i].P2PV2Bootstrapper())
	}

	sourceValueBridge := &client.BridgeTypeAttributes{
		Name:        "bridge-mockadapter",
		URL:         fmt.Sprintf("%s/%s", mockUrl, "five"),
		RequestData: "{}",
	}

	// Setting up job specs
	for nIdx, n := range cc.ChainlinkNodes {
		if nIdx == 0 {
			continue
		}
		_, err := n.CreateBridge(sourceValueBridge)
		if err != nil {
			return err
		}
		relayConfig := job.JSONConfig{
			"nodeName": bootstrapRelayConfig["nodeName"],
			"chainID":  bootstrapRelayConfig["chainID"],
		}

		oracleSpec = job.OCR2OracleSpec{
			ContractID:                  ocrControllerAddress,
			Relay:                       relay.Cosmos,
			RelayConfig:                 relayConfig,
			PluginType:                  "median",
			OCRKeyBundleID:              null.StringFrom(cc.NodeKeys[nIdx].OCR2Key.Data.ID),
			TransmitterID:               null.StringFrom(cc.NodeKeys[nIdx].TXKey.Data.ID),
			P2PV2Bootstrappers:          pq.StringArray{strings.Join(p2pBootstrappers, ",")},
			ContractConfigConfirmations: 1, // don't wait for confirmation on devnet
			PluginConfig: job.JSONConfig{
				"juelsPerFeeCoinSource": juelsPerFeeCoinSource,
			},
		}

		jobSpec = &client.OCR2TaskJobSpec{
			Name:              fmt.Sprintf("starknet-OCRv2-%d-%s", nIdx, uuid.NewString()),
			JobType:           "offchainreporting2",
			OCR2OracleSpec:    oracleSpec,
			ObservationSource: observationSource,
		}
		_, _, err = n.CreateJob(jobSpec)
		if err != nil {
			return err
		}
	}
	return nil
}
