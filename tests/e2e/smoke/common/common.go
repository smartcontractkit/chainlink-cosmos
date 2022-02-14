package common

import (
	"encoding/json"
	"math/big"
	"os"
	"time"

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

// OCRv2State OCR test state
type OCRv2State struct {
	Env            *environment.Environment
	Addresses      *ContractsAddresses
	MockServer     *client.MockserverClient
	Nodes          []client.Chainlink
	Nets           *client.Networks
	LinkToken      *e2e.LinkToken
	BAC            *e2e.AccessController
	RAC            *e2e.AccessController
	Flags          *e2e.OCRv2Flags
	OCR2           *e2e.OCRv2
	Validator      *e2e.OCRv2Validator
	OCR2Proxy      *e2e.OCRv2Proxy
	ValidatorProxy *e2e.OCRv2Proxy
	OCConfig       contracts.OffChainAggregatorV2Config
	NodeKeysBundle []common.NodeKeysBundle
	Transmitters   []string
	RoundsFound    int
	LastRoundTime  time.Time
	Err            error
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

// DeployCluster deploys OCR cluster with or without contracts
func (m *OCRv2State) DeployCluster(nodes int, stateful bool) {
	m.DeployEnv(nodes, stateful)
	m.SetupClients()
	if m.Nets.Default.ContractsDeployed() {
		err := m.LoadContracts()
		Expect(err).ShouldNot(HaveOccurred())
		return
	}
	m.DeployContracts()
	err := m.DumpContracts()
	Expect(err).ShouldNot(HaveOccurred())
	m.CreateJobs()
}

// DeployEnv deploys and connects OCR environment
func (m *OCRv2State) DeployEnv(nodes int, stateful bool) {
	m.Env, m.Err = environment.DeployOrLoadEnvironment(
		e2e.NewChainlinkTerraEnv(nodes, stateful),
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
		e2e.ClientInitFunc(),
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
func (m *OCRv2State) DeployContracts() {
	m.OCConfig, m.NodeKeysBundle, m.Err = common.DefaultOffChainConfigParamsFromNodes(m.Nodes)
	Expect(m.Err).ShouldNot(HaveOccurred())
	cd := e2e.NewTerraContractDeployer(m.Nets.Default)
	Expect(m.Err).ShouldNot(HaveOccurred())

	m.LinkToken, m.Err = cd.DeployLinkTokenContract()
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Err = common.FundOracles(m.Nets.Default, m.NodeKeysBundle, big.NewFloat(5e12))
	Expect(m.Err).ShouldNot(HaveOccurred())

	m.BAC, m.Err = cd.DeployOCRv2AccessController()
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.RAC, m.Err = cd.DeployOCRv2AccessController()
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.OCR2, m.Err = cd.DeployOCRv2(m.BAC.Address(), m.RAC.Address(), m.LinkToken.Address())
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Flags, m.Err = cd.DeployOCRv2Flags(m.BAC.Address(), m.RAC.Address())
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Validator, m.Err = cd.DeployOCRv2Validator(uint32(80000), m.Flags.Address())
	Expect(m.Err).ShouldNot(HaveOccurred())

	m.Err = m.OCR2.SetBilling(uint64(2e5), uint64(1), uint64(1), "1", m.BAC.Address())
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Transmitters, m.Err = m.OCR2.SetOffChainConfig(m.OCConfig)
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Err = m.OCR2.SetPayees(m.Transmitters)
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Err = m.OCR2.SetValidatorConfig(uint64(2e18), m.Validator.Address())
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.OCR2Proxy, m.Err = cd.DeployOCRv2Proxy(m.OCR2.Address())
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.ValidatorProxy, m.Err = cd.DeployOCRv2Proxy(m.Validator.Address())
	Expect(m.Err).ShouldNot(HaveOccurred())
}

// CreateJobs creating OCR jobs and EA stubs
func (m *OCRv2State) CreateJobs() {
	m.Err = m.MockServer.SetValuePath("/variable", 5)
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Err = m.MockServer.SetValuePath("/juels", 1)
	Expect(m.Err).ShouldNot(HaveOccurred())
	m.Err = common.CreateJobs(m.OCR2.Address(), m.Nodes, m.NodeKeysBundle, m.MockServer)
	Expect(m.Err).ShouldNot(HaveOccurred())
}

// LoadContracts loads contracts if they are already deployed
func (m *OCRv2State) LoadContracts() error {
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
	m.OCR2 = &e2e.OCRv2{
		Client: m.Nets.Default.(*e2e.TerraLCDClient),
		Addr:   accAddr,
	}
	return nil
}

// DumpContracts dumps contracts to a file
func (m *OCRv2State) DumpContracts() error {
	s := &ContractsAddresses{OCR: m.OCR2.Address()}
	d, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(ContractsStateFile, d, os.ModePerm)
}

// ValidateNoRoundsAfter validate to rounds are present after some point in time
func (m *OCRv2State) ValidateNoRoundsAfter(chaosStartTime time.Time) {
	m.RoundsFound = 0
	m.LastRoundTime = chaosStartTime
	Consistently(func(g Gomega) {
		_, timestamp, _, err := m.OCR2.GetLatestRoundData()
		g.Expect(err).ShouldNot(HaveOccurred())
		roundTime := time.Unix(int64(timestamp), 0)
		g.Expect(roundTime.Before(m.LastRoundTime)).Should(BeTrue())
	}, NewRoundCheckTimeout, NewRoundCheckPollInterval).Should(Succeed())
}

// ValidateRoundsAfter validates there are new rounds after some point in time
func (m *OCRv2State) ValidateRoundsAfter(chaosStartTime time.Time, rounds int) {
	m.RoundsFound = 0
	m.LastRoundTime = chaosStartTime
	Eventually(func(g Gomega) {
		answer, timestamp, roundID, err := m.OCR2.GetLatestRoundData()
		g.Expect(err).ShouldNot(HaveOccurred())
		roundTime := time.Unix(int64(timestamp), 0)
		g.Expect(roundTime.After(m.LastRoundTime)).Should(BeTrue())
		m.RoundsFound++
		m.LastRoundTime = roundTime
		log.Debug().
			Uint64("RoundID", roundID).
			Int("RoundFound", m.RoundsFound).
			Interface("Answer", answer).
			Time("Time", roundTime).
			Msg("OCR Round")
		g.Expect(m.RoundsFound).Should(Equal(rounds))
	}, NewRoundCheckTimeout, NewRoundCheckPollInterval).Should(Succeed())
}
