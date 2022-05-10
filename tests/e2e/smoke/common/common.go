package common

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/smartcontractkit/chainlink-testing-framework/blockchain"

	"github.com/neilotoole/errgroup"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/common"
	"github.com/smartcontractkit/helmenv/environment"
	"github.com/smartcontractkit/helmenv/tools"
	"github.com/smartcontractkit/chainlink-testing-framework/client"
	"github.com/smartcontractkit/terra.go/msg"
)

const (
	// ContractsStateFile JSON file to store addresses of already deployed contracts
	ContractsStateFile = "contracts-chaos-state.json"
	// NewRoundCheckTimeout how long to await a new round
	NewRoundCheckTimeout = 3 * time.Minute
	// NewSoakRoundCheckTimeout  how long to await soak tests results
	NewSoakRoundCheckTimeout = 3 * time.Hour
	// NewRoundCheckPollInterval new round check interval
	NewRoundCheckPollInterval = 1 * time.Second
	// SourceChangeInterval EA value change interval
	SourceChangeInterval = 1250 * time.Millisecond
	// ChaosAwaitingApply time to wait for chaos experiment to apply
	ChaosAwaitingApply = 60 * time.Second
	// ChaosGroupFaulty Group of faulty nodes, even if they fail OCR must work
	ChaosGroupFaulty = "chaosGroupFaulty"
	// ChaosGroupYellow if nodes from that group fail we may not work while some experiments are going
	// but after experiment it must recover
	ChaosGroupYellow = "chaosGroupYellow"
	// ChaosGroupBootstrap only bootstrap node
	ChaosGroupBootstrap = "chaosGroupBootstrap"
	// ChaosGroupOracles only oracles except bootstrap
	ChaosGroupOracles = "chaosGroupOracles"
	// ChaosGroupOraclesMinusOne all oracles except one
	ChaosGroupOraclesMinusOne = "chaosGroupOraclesMinusOne"
	// ChaosGroupLeftHalf an equal half of all nodes
	ChaosGroupLeftHalf = "chaosGroupLeftHalf"
	// ChaosGroupRightHalf an equal half of all nodes
	ChaosGroupRightHalf = "chaosGroupRightHalf"
	// ChaosGroupOnline a group of nodes that are working
	ChaosGroupOnline = "chaosGroupOnline"
	// UntilStop some chaos experiments doesn't respect absence of duration and got recovered immediately, so we enforce duration
	UntilStop = 666 * time.Hour
)

type Contracts struct {
	LinkToken      *e2e.LinkToken
	BAC            *e2e.AccessController
	RAC            *e2e.AccessController
	Flags          *e2e.OCRv2Flags
	OCR2           *e2e.OCRv2
	Validator      *e2e.OCRv2Validator
	OCR2Proxy      *e2e.OCRv2Proxy
	ValidatorProxy *e2e.OCRv2Proxy
}

// ContractsAddresses deployed contract addresses
type ContractsAddresses struct {
	OCR       string `json:"ocr"`
	LinkToken string `json:"link"`
	BAC       string `json:"bac"`
	RAC       string `json:"rac"`
	Flags     string `json:"flags"`
	Validator string `json:"validator"`
}

// OCRv2State OCR test state
type OCRv2State struct {
	Mu                 *sync.Mutex
	Env                *environment.Environment
	Addresses          *ContractsAddresses
	MockServer         *client.MockserverClient
	Nodes              []client.Chainlink
	Nets               *blockchain.Networks
	Contracts          []Contracts
	ContractsNodeSetup map[int]*common.ContractNodeInfo
	NodeKeysBundle     []common.NodeKeysBundle
	RoundsFound        int
	LastRoundTime      map[string]time.Time
	Err                error
}

func NewOCRv2State(contracts int, nodes int) *OCRv2State {
	state := &OCRv2State{
		Mu:                 &sync.Mutex{},
		LastRoundTime:      make(map[string]time.Time),
		ContractsNodeSetup: make(map[int]*common.ContractNodeInfo),
	}
	for i := 0; i < contracts; i++ {
		state.ContractsNodeSetup[i] = &common.ContractNodeInfo{
			OCR2Address:    "",
			NodesIdx:       []int{},
			Nodes:          []client.Chainlink{},
			NodeKeysBundle: []common.NodeKeysBundle{},
			BridgeInfos:    []common.BridgeInfo{},
		}
		state.ContractsNodeSetup[i].BootstrapNodeIdx = 0
		for n := 1; n < nodes; n++ {
			state.ContractsNodeSetup[i].NodesIdx = append(state.ContractsNodeSetup[i].NodesIdx, n)
		}
	}
	return state
}

// DeployCluster deploys OCR cluster with or without contracts
func (m *OCRv2State) DeployCluster(nodes int, blockTime string, stateful bool, contractsDir string) {
	m.DeployEnv(nodes, blockTime, stateful)
	m.SetupClients()
	m.DeployContracts(contractsDir)
	err := m.DumpContracts()
	Expect(err).ShouldNot(HaveOccurred())
	m.CreateJobs()
}

// DeployEnv deploys and connects OCR environment
func (m *OCRv2State) DeployEnv(nodes int, blockTime string, stateful bool) {
	m.Env, m.Err = environment.DeployOrLoadEnvironment(
		e2e.NewChainlinkTerraEnv(nodes, blockTime, stateful),
		tools.ChartsRoot,
	)
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Err = m.Env.ConnectAll()
	Expect(m.Err).ShouldNot(HaveOccurred())
}

// SetupClients setting up clients
func (m *OCRv2State) SetupClients() {
	networkRegistry := blockchain.NewDefaultNetworkRegistry()
	networkRegistry.RegisterNetwork(
		"terra",
		e2e.ClientInitFunc(len(m.ContractsNodeSetup)),
		e2e.ClientURLSFunc(),
	)
	m.Nets, m.Err = networkRegistry.GetNetworks(m.Env)
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.MockServer, m.Err = client.ConnectMockServer(m.Env)
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Nodes, m.Err = client.ConnectChainlinkNodes(m.Env)
	Expect(m.Err).ShouldNot(HaveOccurred())
}

func (m *OCRv2State) initializeNodesInContractsMap() {
	for i := 0; i < len(m.ContractsNodeSetup); i++ {
		for _, nodeIndex := range m.ContractsNodeSetup[i].NodesIdx {
			m.ContractsNodeSetup[i].Nodes = append(m.ContractsNodeSetup[i].Nodes, m.Nodes[nodeIndex])
			m.ContractsNodeSetup[i].NodeKeysBundle = append(m.ContractsNodeSetup[i].NodeKeysBundle, m.NodeKeysBundle[nodeIndex])
		}
		m.ContractsNodeSetup[i].BootstrapNode = m.Nodes[m.ContractsNodeSetup[i].BootstrapNodeIdx]
		m.ContractsNodeSetup[i].BootstrapNodeKeysBundle = m.NodeKeysBundle[m.ContractsNodeSetup[i].BootstrapNodeIdx]
	}
}

// DeployContracts deploys contracts
func (m *OCRv2State) DeployContracts(contractsDir string) {
	defaultNetwork := m.Nets.Default

	m.NodeKeysBundle, m.Err = common.CreateNodeKeysBundle(m.Nodes)
	Expect(m.Err).ShouldNot(HaveOccurred())

	m.Err = common.FundOracles(defaultNetwork, m.NodeKeysBundle, big.NewFloat(5e8))
	Expect(m.Err).ShouldNot(HaveOccurred())

	c := defaultNetwork.GetClients()[0]
	cd := e2e.NewTerraContractDeployer(c)
	lt, err := cd.DeployLinkTokenContract()
	Expect(err).ShouldNot(HaveOccurred())

	m.initializeNodesInContractsMap()
	g := errgroup.Group{}
	for i := 0; i < len(m.ContractsNodeSetup); i++ {
		i := i
		g.Go(func() error {
			defer ginkgo.GinkgoRecover()
			c := defaultNetwork.GetClients()[i]
			cd := e2e.NewTerraContractDeployer(c)

			bac, err := cd.DeployOCRv2AccessController(contractsDir)
			Expect(err).ShouldNot(HaveOccurred())
			rac, err := cd.DeployOCRv2AccessController(contractsDir)
			Expect(err).ShouldNot(HaveOccurred())
			ocr2, err := cd.DeployOCRv2(bac.Address(), rac.Address(), lt.Address(), contractsDir)
			Expect(err).ShouldNot(HaveOccurred())
			flags, err := cd.DeployOCRv2Flags(bac.Address(), rac.Address(), contractsDir)
			Expect(err).ShouldNot(HaveOccurred())
			validator, err := cd.DeployOCRv2Validator(uint32(80000), flags.Address(), contractsDir)
			Expect(err).ShouldNot(HaveOccurred())

			err = ocr2.SetBilling(uint64(2e5), uint64(1), uint64(1), "1", bac.Address())
			Expect(err).ShouldNot(HaveOccurred())

			ocConfig, err := common.OffChainConfigParamsFromNodes(m.ContractsNodeSetup[i].Nodes, m.ContractsNodeSetup[i].NodeKeysBundle)
			Expect(err).ShouldNot(HaveOccurred())

			_, err = ocr2.SetOffChainConfig(ocConfig)
			Expect(err).ShouldNot(HaveOccurred())

			err = ocr2.SetValidatorConfig(uint64(2e18), validator.Address())
			Expect(err).ShouldNot(HaveOccurred())
			ocrProxy, err := cd.DeployOCRv2Proxy(ocr2.Address(), contractsDir)
			Expect(err).ShouldNot(HaveOccurred())
			validatorProxy, err := cd.DeployOCRv2Proxy(validator.Address(), contractsDir)
			Expect(err).ShouldNot(HaveOccurred())

			m.Mu.Lock()
			m.Contracts = append(m.Contracts, Contracts{
				LinkToken:      lt,
				BAC:            bac,
				RAC:            rac,
				Flags:          flags,
				OCR2:           ocr2,
				Validator:      validator,
				OCR2Proxy:      ocrProxy,
				ValidatorProxy: validatorProxy,
			})
			m.Mu.Unlock()
			return nil
		})
	}
	Expect(g.Wait()).ShouldNot(HaveOccurred())
	for i := 0; i < len(m.ContractsNodeSetup); i++ {
		m.ContractsNodeSetup[i].OCR2Address = m.Contracts[i].OCR2.Address()
	}
}

func (m *OCRv2State) SetAllAdapterResponsesToTheSameValue(response int) {
	for i := 0; i < len(m.ContractsNodeSetup); i++ {
		for _, node := range m.ContractsNodeSetup[i].Nodes {
			nodeContractPairID, err := common.BuildNodeContractPairID(node, m.ContractsNodeSetup[i].OCR2Address)
			Expect(err).ShouldNot(HaveOccurred())
			path := fmt.Sprintf("/%s", nodeContractPairID)
			m.Err = m.MockServer.SetValuePath(path, response)
			Expect(m.Err).ShouldNot(HaveOccurred())
		}
	}
}

// CreateJobs creating OCR jobs and EA stubs
func (m *OCRv2State) CreateJobs() {
	m.SetAllAdapterResponsesToTheSameValue(5)
	err := m.MockServer.SetValuePath("/juels", 1)
	Expect(err).ShouldNot(HaveOccurred())
	err = common.CreateTerraChainAndNode(m.Nodes)
	Expect(err).ShouldNot(HaveOccurred())

	err = common.CreateBridges(m.ContractsNodeSetup, m.MockServer)
	Expect(err).ShouldNot(HaveOccurred())
	g := errgroup.Group{}
	for i := 0; i < len(m.ContractsNodeSetup); i++ {
		i := i
		g.Go(func() error {
			defer ginkgo.GinkgoRecover()
			m.Err = common.CreateJobs(m.ContractsNodeSetup[i])
			Expect(m.Err).ShouldNot(HaveOccurred())
			return nil
		})
	}
	Expect(g.Wait()).ShouldNot(HaveOccurred())
}

// LoadContracts loads contracts if they are already deployed
func (m *OCRv2State) LoadContracts() error {
	for range m.ContractsNodeSetup {
		d, err := os.ReadFile(ContractsStateFile)
		if err != nil {
			return err
		}
		var contractsState *ContractsAddresses
		if err = json.Unmarshal(d, &contractsState); err != nil {
			return err
		}
		accAddr, err := msg.AccAddressFromBech32(contractsState.OCR)
		if err != nil {
			return err
		}
		m.Contracts = append(m.Contracts, Contracts{OCR2: &e2e.OCRv2{
			Client: m.Nets.Default.(*e2e.TerraLCDClient),
			Addr:   accAddr,
		}})
	}
	return nil
}

func (m *OCRv2State) UpdateChainlinkVersion(image string, version string) {
	chart, err := m.Env.Charts.Get("chainlink")
	Expect(err).ShouldNot(HaveOccurred())
	chart.Values["chainlink"] = map[string]interface{}{
		"image": map[string]interface{}{
			"image":   image,
			"version": version,
		},
	}
	err = chart.Upgrade()
	Expect(err).ShouldNot(HaveOccurred())
	err = m.Env.ConnectAll()
	Expect(err).ShouldNot(HaveOccurred())
}

// DumpContracts dumps contracts to a file
func (m *OCRv2State) DumpContracts() error {
	//s := &ContractsAddresses{OCR: m.OCR2.Address()}
	//d, err := json.Marshal(s)
	//if err != nil {
	//	return err
	//}
	//return os.WriteFile(ContractsStateFile, d, os.ModePerm)
	return nil
}

// ValidateNoRoundsAfter validate to rounds are present after some point in time
func (m *OCRv2State) ValidateNoRoundsAfter(startTime time.Time) {
	m.RoundsFound = 0
	for _, c := range m.Contracts {
		m.LastRoundTime[c.OCR2.Address()] = startTime
	}
	Consistently(func(g Gomega) {
		for _, c := range m.Contracts {
			_, timestamp, _, err := c.OCR2.GetLatestRoundData()
			g.Expect(err).ShouldNot(HaveOccurred())
			roundTime := time.Unix(int64(timestamp), 0)
			g.Expect(roundTime.Before(m.LastRoundTime[c.OCR2.Address()])).Should(BeTrue())
		}
	}, NewRoundCheckTimeout, NewRoundCheckPollInterval).Should(Succeed())
}

type Answer struct {
	Answer    uint64
	Timestamp uint64
	RoundID   uint64
	Error     error
}

func (m *OCRv2State) ValidateAllRounds(startTime time.Time, timeout time.Duration, rounds int, proxy bool) {
	m.RoundsFound = 0
	for _, c := range m.Contracts {
		m.LastRoundTime[c.OCR2.Address()] = startTime
	}
	roundsFound := 0
	Eventually(func(g Gomega) {
		answers := make(map[string]*Answer)
		for _, c := range m.Contracts {
			var answer, timestamp, roundID uint64
			var err error
			if proxy {
				answer, timestamp, roundID, err = c.OCR2Proxy.GetLatestRoundData()
			} else {
				answer, timestamp, roundID, err = c.OCR2.GetLatestRoundData()
			}
			answers[c.OCR2.Address()] = &Answer{Answer: answer, Timestamp: timestamp, RoundID: roundID, Error: err}
		}
		for ci, a := range answers {
			log.Debug().Str("Contract", ci).Interface("Answer", a).Msg("Answer found")
			answerTime := time.Unix(int64(a.Timestamp), 0)
			if answerTime.After(m.LastRoundTime[ci]) {
				m.LastRoundTime[ci] = answerTime
				roundsFound++
				log.Debug().Int("RoundsFound", roundsFound).Send()
			}
		}
		g.Expect(roundsFound).To(BeNumerically(">=", rounds*len(m.Contracts)))
	}, timeout, NewRoundCheckPollInterval).Should(Succeed())
}
