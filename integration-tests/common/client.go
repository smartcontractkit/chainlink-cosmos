package common

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"gopkg.in/guregu/null.v4"

	"github.com/smartcontractkit/chainlink-cosmos/pkg/cosmos/params"
	"github.com/smartcontractkit/chainlink-testing-framework/k8s/environment"
	"github.com/smartcontractkit/chainlink/integration-tests/client"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/relay"
)

type ChainlinkClient struct {
	bech32Prefix   string
	ChainlinkNodes []*client.ChainlinkClient
	NodeKeys       []client.NodeKeysBundle
	bTypeAttr      *client.BridgeTypeAttributes
	bootstrapPeers []client.P2PData
}

// TODO: Remove env. See https://github.com/smartcontractkit/chainlink-cosmos/pull/350#discussion_r1298071289
// CreateKeys Creates node keys and defines chain and nodes for each node
func NewChainlinkClient(env *environment.Environment, nodeName string, chainId string, tendermintURL string, bech32Prefix string) (*ChainlinkClient, error) {
	nodes, err := connectChainlinkNodes(env)
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

	return &ChainlinkClient{
		bech32Prefix:   bech32Prefix,
		ChainlinkNodes: nodes,
		NodeKeys:       nodeKeys,
	}, nil
}

func (cc *ChainlinkClient) GetNodeAddresses() []string {
	var addresses []string
	for _, nodeKey := range cc.NodeKeys {
		addresses = append(addresses, mustCreateBech32Address(nodeKey.TXKey.Data.Attributes.PublicKey, cc.bech32Prefix))
	}
	return addresses
}

func mustCreateBech32Address(pubKey, accountPrefix string) string {
	bech32Addr, err := params.CreateBech32Address(pubKey, accountPrefix)
	if err != nil {
		panic(err)
	}
	return bech32Addr
}

func (cc *ChainlinkClient) LoadOCR2Config(proposalId string) (*OCR2Config, error) {
	var offChainKeys []string
	var onChainKeys []string
	var peerIds []string
	var cfgKeys []string
	for _, key := range cc.NodeKeys {
		offChainKeys = append(offChainKeys, key.OCR2Key.Data.Attributes.OffChainPublicKey)
		peerIds = append(peerIds, key.PeerID)
		onChainKeys = append(onChainKeys, key.OCR2Key.Data.Attributes.OnChainPublicKey)
		cfgKeys = append(cfgKeys, key.OCR2Key.Data.Attributes.ConfigPublicKey)
	}
	var payload = TestOCR2Config
	payload.ProposalId = proposalId
	payload.Signers = onChainKeys
	addresses := cc.GetNodeAddresses()
	payload.Transmitters = addresses
	payload.Payees = addresses // Set payees to same addresses as transmitters
	payload.OffchainConfig.OffchainPublicKeys = offChainKeys
	payload.OffchainConfig.PeerIds = peerIds
	payload.OffchainConfig.ConfigPublicKeys = cfgKeys
	return &payload, nil
}

// CreateJobsForContract Creates and sets up the boostrap jobs as well as OCR jobs
func (cc *ChainlinkClient) CreateJobsForContract(chainId, nodeName, p2pPort, mockUrl string, juelsPerFeeCoinSource string, ocrControllerAddress string) error {
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
		"nodeName": fmt.Sprintf("\"%s\"", nodeName),
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

	_, err := cc.ChainlinkNodes[0].MustCreateJob(jobSpec)
	if err != nil {
		return err
	}

	var p2pBootstrappers []string

	for i := range cc.bootstrapPeers {
		p2pBootstrappers = append(p2pBootstrappers, cc.bootstrapPeers[i].P2PV2Bootstrapper())
	}

	sourceValueBridge := &client.BridgeTypeAttributes{
		Name:        "mockserver-bridge",
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
			TransmitterID:               null.StringFrom(cc.NodeKeys[nIdx].TXKey.Data.Attributes.PublicKey),
			P2PV2Bootstrappers:          pq.StringArray{strings.Join(p2pBootstrappers, ",")},
			ContractConfigConfirmations: 1, // don't wait for confirmation on devnet
			PluginConfig: job.JSONConfig{
				"juelsPerFeeCoinSource": juelsPerFeeCoinSource,
			},
		}

		jobSpec = &client.OCR2TaskJobSpec{
			Name:              fmt.Sprintf("cosmos-OCRv2-%d-%s", nIdx, uuid.NewString()),
			JobType:           "offchainreporting2",
			OCR2OracleSpec:    oracleSpec,
			ObservationSource: client.ObservationSourceSpecBridge(sourceValueBridge),
		}

		_, err = n.MustCreateJob(jobSpec)
		if err != nil {
			return err
		}
	}
	return nil
}

// connectChainlinkNodes creates a chainlink client for each node in the environment
// This is a non k8s version of the function in chainlink_k8s.go
// https://github.com/smartcontractkit/chainlink/blob/cosmos-test-keys/integration-tests/client/chainlink_k8s.go#L77
func connectChainlinkNodes(e *environment.Environment) ([]*client.ChainlinkClient, error) {
	var clients []*client.ChainlinkClient
	for _, nodeDetails := range e.ChainlinkNodeDetails {
		c, err := client.NewChainlinkClient(&client.ChainlinkConfig{
			URL:        nodeDetails.LocalIP,
			Email:      "notreal@fakeemail.ch",
			Password:   "fj293fbBnlQ!f9vNs",
			InternalIP: parseHostname(nodeDetails.InternalIP),
		}, log.Logger)
		if err != nil {
			return nil, err
		}
		log.Debug().
			Str("URL", c.Config.URL).
			Str("Internal IP", c.Config.InternalIP).
			Str("Chart Name", nodeDetails.ChartName).
			Str("Pod Name", nodeDetails.PodName).
			Msg("Connected to Chainlink node")
		clients = append(clients, c)
	}
	return clients, nil
}

func parseHostname(s string) string {
	r := regexp.MustCompile(`://(?P<Host>.*):`)
	return r.FindStringSubmatch(s)[1]
}
