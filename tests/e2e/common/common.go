package common

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"

	"github.com/smartcontractkit/integrations-framework/client"
	"github.com/smartcontractkit/integrations-framework/contracts"
	"github.com/smartcontractkit/libocr/offchainreporting2/confighelper"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"golang.org/x/crypto/curve25519"
)

// TODO: those should be moved as a common part of integrations-framework

const (
	ChainName = "terra"
)

// Those functions may be common with another chains and should be moved to another lib

type NodeKeysBundle struct {
	PeerID  string
	OCR2Key *client.OCR2Key
	TXKey   *client.TxKey
}

// OCR2 keys are in format OCR2<key_type>_<network>_<key>
func stripKeyPrefix(key string) string {
	chunks := strings.Split(key, "_")
	if len(chunks) == 3 {
		return chunks[2]
	}
	return key
}

func createNodeKeys(nodes []client.Chainlink) ([]NodeKeysBundle, error) {
	nkb := make([]NodeKeysBundle, 0)
	for _, n := range nodes {
		p2pkeys, err := n.ReadP2PKeys()
		if err != nil {
			return nil, err
		}

		peerID := p2pkeys.Data[0].Attributes.PeerID
		txKey, err := n.CreateTxKey(ChainName)
		if err != nil {
			return nil, err
		}
		ocrKey, err := n.CreateOCR2Key(ChainName)
		if err != nil {
			return nil, err
		}
		nkb = append(nkb, NodeKeysBundle{
			PeerID:  peerID,
			OCR2Key: ocrKey,
			TXKey:   txKey,
		})
	}
	return nkb, nil
}

func createOracleIdentities(nkb []NodeKeysBundle) ([]confighelper.OracleIdentityExtra, error) {
	oracleIdentities := make([]confighelper.OracleIdentityExtra, 0)
	for _, nodeKeys := range nkb {
		offChainPubKey, err := hex.DecodeString(stripKeyPrefix(nodeKeys.OCR2Key.Data.Attributes.OffChainPublicKey))
		if err != nil {
			return nil, err
		}
		onChainPubKey, err := hex.DecodeString(stripKeyPrefix(nodeKeys.OCR2Key.Data.Attributes.OnChainPublicKey))
		if err != nil {
			return nil, err
		}
		cfgPubKeyTemp, err := hex.DecodeString(stripKeyPrefix(nodeKeys.OCR2Key.Data.Attributes.ConfigPublicKey))
		if err != nil {
			return nil, err
		}
		cfgPubKeyBytes := [curve25519.PointSize]byte{}
		copy(cfgPubKeyBytes[:], cfgPubKeyTemp)
		oracleIdentities = append(oracleIdentities, confighelper.OracleIdentityExtra{
			OracleIdentity: confighelper.OracleIdentity{
				OffchainPublicKey: offChainPubKey,
				OnchainPublicKey:  onChainPubKey,
				PeerID:            nodeKeys.PeerID,
				TransmitAccount:   types.Account(nodeKeys.TXKey.Data.Attributes.PublicKey),
			},
			ConfigEncryptionPublicKey: cfgPubKeyBytes,
		})
	}
	// program sorts oracles (need to pre-sort to allow correct onchainConfig generation)
	sort.Slice(oracleIdentities, func(i, j int) bool {
		return bytes.Compare(oracleIdentities[i].OracleIdentity.OnchainPublicKey, oracleIdentities[j].OracleIdentity.OnchainPublicKey) < 0
	})
	return oracleIdentities, nil
}

func FundOracles(c client.BlockchainClient, nkb []NodeKeysBundle, amount *big.Float) error {
	for _, nk := range nkb {
		addr := nk.TXKey.Data.Attributes.PublicKey
		if err := c.Fund(addr, amount); err != nil {
			return err
		}
	}
	return nil
}

// DefaultOffChainConfigParamsFromNodes collects OCR2 keys and creates contracts.OffChainAggregatorV2Config
func DefaultOffChainConfigParamsFromNodes(nodes []client.Chainlink) (contracts.OffChainAggregatorV2Config, []NodeKeysBundle, error) {
	nkb, err := createNodeKeys(nodes)
	if err != nil {
		return contracts.OffChainAggregatorV2Config{}, nil, err
	}
	oi, err := createOracleIdentities(nkb[1:])
	if err != nil {
		return contracts.OffChainAggregatorV2Config{}, nil, err
	}
	s := make([]int, 0)
	for range nodes[1:] {
		s = append(s, 1)
	}
	faultyNodes := 0
	if len(nodes[1:]) > 1 {
		faultyNodes = len(nkb[1:])/3 - 1
	}
	if faultyNodes == 0 {
		faultyNodes = 1
	}
	log.Warn().Int("Nodes", faultyNodes).Msg("Faulty nodes")
	return contracts.OffChainAggregatorV2Config{
		DeltaProgress: 2 * time.Second,
		DeltaResend:   5 * time.Second,
		DeltaRound:    1 * time.Second,
		DeltaGrace:    500 * time.Millisecond,
		DeltaStage:    10 * time.Second,
		RMax:          3,
		S:             s,
		Oracles:       oi,
		ReportingPluginConfig: median.OffchainConfig{
			AlphaReportPPB: uint64(0),
			AlphaAcceptPPB: uint64(0),
		}.Encode(),
		MaxDurationQuery:                        0,
		MaxDurationObservation:                  500 * time.Millisecond,
		MaxDurationReport:                       500 * time.Millisecond,
		MaxDurationShouldAcceptFinalizedReport:  500 * time.Millisecond,
		MaxDurationShouldTransmitAcceptedReport: 500 * time.Millisecond,
		F:                                       faultyNodes,
		OnchainConfig:                           []byte{},
	}, nkb, nil
}

func ImitateSource(mockServer *client.MockserverClient, changeInterval time.Duration, min int, max int) {
	go func() {
		for {
			_ = mockServer.SetValuePath("/variable", min)
			time.Sleep(changeInterval)
			_ = mockServer.SetValuePath("/variable", max)
			time.Sleep(changeInterval)
		}
	}()
}

func CreateJobs(ocr2Addr string, nodes []client.Chainlink, nkb []NodeKeysBundle, mock *client.MockserverClient) error {
	bootstrapPeers := []client.P2PData{
		{
			RemoteIP:   nodes[0].RemoteIP(),
			RemotePort: "6690",
			PeerID:     nkb[0].PeerID,
		},
	}
	for nIdx, n := range nodes {
		var IsBootstrapPeer bool
		if nIdx == 0 {
			IsBootstrapPeer = true
		}
		sourceValueBridge := client.BridgeTypeAttributes{
			Name:        "variable",
			URL:         fmt.Sprintf("%s/variable", mock.Config.ClusterURL),
			RequestData: "{}",
		}
		observationSource := client.ObservationSourceSpecBridge(sourceValueBridge)
		err := n.CreateBridge(&sourceValueBridge)
		if err != nil {
			return err
		}

		juelsBridge := client.BridgeTypeAttributes{
			Name:        "juels",
			URL:         fmt.Sprintf("%s/juels", mock.Config.ClusterURL),
			RequestData: "{}",
		}
		juelsSource := client.ObservationSourceSpecBridge(juelsBridge)
		err = n.CreateBridge(&juelsBridge)
		if err != nil {
			return err
		}
		_, err = n.CreateTerraChain(&client.TerraChainAttributes{ChainID: "localterra"})
		if err != nil {
			return err
		}
		relayConfig := map[string]string{
			"nodeType":      "terra",
			"tendermintURL": "http://terrad:26657",
			"fcdURL":        "http://fcd-api:3060",
			"chainID":       "localterra",
		}
		if _, err = n.CreateTerraNode(&client.TerraNodeAttributes{
			Name:          "terra",
			TerraChainID:  relayConfig["chainID"],
			TendermintURL: relayConfig["tendermintURL"],
			FCDURL:        relayConfig["fcdURL"],
		}); err != nil {
			return err
		}
		jobSpec := &client.OCR2TaskJobSpec{
			Name:                  fmt.Sprintf("terra-OCRv2-%d-%s", nIdx, uuid.NewV4().String()),
			ContractID:            ocr2Addr,
			Relay:                 ChainName,
			RelayConfig:           relayConfig,
			P2PPeerID:             nkb[nIdx].PeerID,
			P2PBootstrapPeers:     bootstrapPeers,
			IsBootstrapPeer:       IsBootstrapPeer,
			OCRKeyBundleID:        nkb[nIdx].OCR2Key.Data.ID,
			TransmitterID:         nkb[nIdx].TXKey.Data.ID,
			ObservationSource:     observationSource,
			JuelsPerFeeCoinSource: juelsSource,
		}
		if _, err = n.CreateJob(jobSpec); err != nil {
			return err
		}
	}
	return nil
}

// GetDefaultGauntletConfig gets  the default config gauntlet will need to start making commands
// 	against the environment
func GetDefaultGauntletConfig(nodeUrl *url.URL) map[string]string {
	networkConfig := map[string]string{
		"NETWORK":           "localterra",
		"NODE_URL":          nodeUrl.String(),
		"CHAIN_ID":          "localterra",
		"DEFAULT_GAS_PRICE": "1",
		"MNEMONIC":          "satisfy adjust timber high purchase tuition stool faith fine install that you unaware feed domain license impose boss human eager hat rent enjoy dawn",
	}

	return networkConfig
}
