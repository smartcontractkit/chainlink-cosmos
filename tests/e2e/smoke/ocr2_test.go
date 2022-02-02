package smoke_test

import (
	"math/big"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/common"
	"github.com/smartcontractkit/helmenv/environment"
	"github.com/smartcontractkit/helmenv/tools"
	"github.com/smartcontractkit/integrations-framework/actions"
	"github.com/smartcontractkit/integrations-framework/client"
	"github.com/smartcontractkit/integrations-framework/contracts"
)

var _ = Describe("Terra OCRv2 @ocr", func() {
	var (
		e          *environment.Environment
		mockServer *client.MockserverClient
		nodes      []client.Chainlink
		nets       *client.Networks
		lt         *e2e.LinkToken
		bac        *e2e.AccessController
		rac        *e2e.AccessController
		flags      *e2e.OCRv2Flags
		ocr2       *e2e.OCRv2
		validator  *e2e.OCRv2Validator
		ocConfig   contracts.OffChainAggregatorV2Config
		nkb        []common.NodeKeysBundle
		err        error
	)

	BeforeEach(func() {
		By("Deploying the environment", func() {
			e, err = environment.DeployOrLoadEnvironment(
				e2e.NewChainlinkTerraEnv(5),
				tools.ChartsRoot,
			)
			Expect(err).ShouldNot(HaveOccurred())
			err = e.ConnectAll()
			Expect(err).ShouldNot(HaveOccurred())
		})
		By("Setting up client", func() {
			networkRegistry := client.NewNetworkRegistry()
			networkRegistry.RegisterNetwork(
				"terra",
				e2e.ClientInitFunc(),
				e2e.ClientURLSFunc(),
			)
			nets, err = networkRegistry.GetNetworks(e)
			Expect(err).ShouldNot(HaveOccurred())
			mockServer, err = client.ConnectMockServer(e)
			Expect(err).ShouldNot(HaveOccurred())
			nodes, err = client.ConnectChainlinkNodes(e)
			Expect(err).ShouldNot(HaveOccurred())
		})
		By("Deploying contracts", func() {
			ocConfig, nkb, err = common.DefaultOffChainConfigParamsFromNodes(nodes)
			Expect(err).ShouldNot(HaveOccurred())
			cd := e2e.NewTerraContractDeployer(nets.Default)
			Expect(err).ShouldNot(HaveOccurred())

			lt, err = cd.DeployLinkTokenContract()
			Expect(err).ShouldNot(HaveOccurred())
			err = common.FundOracles(nets.Default, nkb, big.NewFloat(5e12))
			Expect(err).ShouldNot(HaveOccurred())

			bac, err = cd.DeployOCRv2AccessController()
			Expect(err).ShouldNot(HaveOccurred())
			rac, err = cd.DeployOCRv2AccessController()
			Expect(err).ShouldNot(HaveOccurred())
			ocr2, err = cd.DeployOCRv2(bac.Address(), rac.Address(), lt.Address())
			Expect(err).ShouldNot(HaveOccurred())
			flags, err = cd.DeployOCRv2Flags(bac.Address(), rac.Address())
			Expect(err).ShouldNot(HaveOccurred())
			validator, err = cd.DeployOCRv2Validator(uint32(80000), flags.Address())
			Expect(err).ShouldNot(HaveOccurred())
			log.Debug().Str("Addr", validator.Address()).Msg("Validator address")

			err = ocr2.SetBilling(uint32(1), uint32(1), bac.Address())
			Expect(err).ShouldNot(HaveOccurred())
			err = ocr2.SetOffChainConfig(ocConfig)
			Expect(err).ShouldNot(HaveOccurred())
		})
		By("Creating jobs", func() {
			err = mockServer.SetValuePath("/variable", 5)
			Expect(err).ShouldNot(HaveOccurred())
			err = mockServer.SetValuePath("/juels", 1)
			Expect(err).ShouldNot(HaveOccurred())
			common.ImitateSource(mockServer, 1*time.Second, 2, 10)
			err = common.CreateJobs(ocr2.Address(), nodes, nkb, mockServer)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("with Terra OCR2", func() {
		It("performs OCR2 round", func() {
			Eventually(func(g Gomega) {
				answer, _, _, err := ocr2.GetLatestRoundData()
				g.Expect(err).ShouldNot(HaveOccurred())
				log.Debug().
					Interface("Answer", answer).
					Msg("OCR Round")
				g.Expect(answer).Should(Equal(uint64(10)))
			}, 3*time.Minute, 1*time.Second).Should(Succeed())
		})
	})

	AfterEach(func() {
		By("Tearing down the environment", func() {
			err = actions.TeardownSuite(e, nil, "logs")
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
