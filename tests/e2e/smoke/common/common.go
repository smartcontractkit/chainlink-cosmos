package common

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/neilotoole/errgroup"

	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/common"
	"github.com/smartcontractkit/helmenv/environment"
	"github.com/smartcontractkit/helmenv/tools"
	"github.com/smartcontractkit/integrations-framework/client"
	"github.com/smartcontractkit/integrations-framework/contracts"
	"github.com/smartcontractkit/terra.go/msg"
)

const (
	// ContractsStateFile JSON file to store addresses of already deployed contracts
	ContractsStateFile = "contracts-chaos-state.json"
	// NewRoundCheckTimeout how long to await a new round
	NewRoundCheckTimeout = 3 * time.Minute
	// NewSoakRoundCheckTimeout  how long to await soak tests results
	NewSoakRoundCheckTimeout = 12 * time.Hour
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

func NewOCRv2State(contracts int) *OCRv2State {
	return &OCRv2State{
		Mu:            &sync.Mutex{},
		ContractsNum:  contracts,
		LastRoundTime: make(map[string]time.Time),
	}
}

// OCRv2State OCR test state
type OCRv2State struct {
	Mu             *sync.Mutex
	Env            *environment.Environment
	Addresses      *ContractsAddresses
	MockServer     *client.MockserverClient
	Nodes          []client.Chainlink
	Nets           *client.Networks
	ContractsNum   int
	Contracts      []Contracts
	OCConfig       contracts.OffChainAggregatorV2Config
	NodeKeysBundle []common.NodeKeysBundle
	Transmitters   []string
	RoundsFound    int
	LastRoundTime  map[string]time.Time
	Err            error
}

// DeployCluster deploys OCR cluster with or without contracts
func (m *OCRv2State) DeployCluster(nodes int, blockTime string, stateful bool, contractsDir string) {
	m.DeployEnv(nodes, blockTime, stateful)
	m.SetupClients()
	if m.Nets.Default.ContractsDeployed() {
		err := m.LoadContracts()
		Expect(err).ShouldNot(HaveOccurred())
		return
	}
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
	networkRegistry := client.NewNetworkRegistry()
	networkRegistry.RegisterNetwork(
		"terra",
		e2e.ClientInitFunc(m.ContractsNum),
		e2e.ClientURLSFunc(),
	)
	m.Nets, m.Err = networkRegistry.GetNetworks(m.Env)
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.MockServer, m.Err = client.ConnectMockServer(m.Env)
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Nodes, m.Err = client.ConnectChainlinkNodes(m.Env)
	Expect(m.Err).ShouldNot(HaveOccurred())
}

// DeployContracts deploys contracts
func (m *OCRv2State) DeployContracts(contractsDir string) {
	defaultNetwork := m.Nets.Default
	m.OCConfig, m.NodeKeysBundle, m.Err = common.DefaultOffChainConfigParamsFromNodes(m.Nodes)
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Err = common.FundOracles(defaultNetwork, m.NodeKeysBundle, big.NewFloat(5e8))
	Expect(m.Err).ShouldNot(HaveOccurred())
	g := errgroup.Group{}
	for i := 0; i < m.ContractsNum; i++ {
		i := i
		g.Go(func() error {
			c := defaultNetwork.GetClients()[i]
			cd := e2e.NewTerraContractDeployer(c)
			lt, err := cd.DeployLinkTokenContract()
			Expect(err).ShouldNot(HaveOccurred())

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
			m.Transmitters, err = ocr2.SetOffChainConfig(m.OCConfig)
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
}

func (m *OCRv2State) SetAllAdapterResponsesToTheSameValue(response int) {
	for _, contract := range m.Contracts {
		for _, node := range m.Nodes {
			nodeContractPairID, err := common.BuildNodeContractPairID(node, contract.OCR2.Address())
			Expect(err).ShouldNot(HaveOccurred())
			path := fmt.Sprintf("/%s", nodeContractPairID)
			m.Err = m.MockServer.SetValuePath(path, response)
			Expect(m.Err).ShouldNot(HaveOccurred())
		}
	}
}

func (m *OCRv2State) SetAllAdapterResponsesToDifferentValues(responses []int) {
	for _, contract := range m.Contracts {
		for nodeIndex, node := range m.Nodes {
			nodeContractPairID, err := common.BuildNodeContractPairID(node, contract.OCR2.Address())
			Expect(err).ShouldNot(HaveOccurred())
			path := fmt.Sprintf("/%s", nodeContractPairID)
			m.Err = m.MockServer.SetValuePath(path, responses[nodeIndex])
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

	ocr2Contracts := []string{}
	for _, contract := range m.Contracts {
		ocr2Contracts = append(ocr2Contracts, contract.OCR2.Address())
	}
	biMap, err := common.CreateBridges(ocr2Contracts, m.Nodes, m.MockServer)
	Expect(err).ShouldNot(HaveOccurred())
	g := errgroup.Group{}
	for i := 0; i < m.ContractsNum; i++ {
		i := i
		g.Go(func() error {
			m.Err = common.CreateJobs(m.Contracts[i].OCR2.Address(), biMap[m.Contracts[i].OCR2.Address()], m.Nodes, m.NodeKeysBundle)
			Expect(m.Err).ShouldNot(HaveOccurred())
			return nil
		})
	}
	Expect(g.Wait()).ShouldNot(HaveOccurred())
}

// LoadContracts loads contracts if they are already deployed
func (m *OCRv2State) LoadContracts() error {
	for i := 0; i < m.ContractsNum; i++ {
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
