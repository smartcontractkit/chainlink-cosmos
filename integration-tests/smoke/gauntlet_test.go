package smoke_test

import (
	"fmt"
	"math/big"
	"net/url"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/smartcontractkit/chainlink-testing-framework/gauntlet"
	"github.com/smartcontractkit/chainlink/integration-tests/actions"

	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/common"
	tc "github.com/smartcontractkit/chainlink-cosmos/integration-tests/smoke/common"
	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/utils"
)

var _ = Describe("Terra Gauntlet @gauntlet", func() {
	var (
		gd    *integration_tests.GauntletDeployer
		state *tc.OCRv2State
	)

	BeforeEach(func() {
		By("Deploying the environment", func() {
			gd = &integration_tests.GauntletDeployer{
				Version: "local",
			}
			state = tc.NewOCRv2State(1, 1)
			state.DeployEnv(1, common.ChainBlockTime, false)
			state.SetupClients()
			state.NodeKeysBundle, state.Err = common.CreateNodeKeysBundle(state.Nodes)
			Expect(state.Err).ShouldNot(HaveOccurred())

			_, state.Err = common.OffChainConfigParamsFromNodes(state.Nodes, state.NodeKeysBundle)
			Expect(state.Err).ShouldNot(HaveOccurred())
		})
		By("Setup Gauntlet", func() {
			cwd, err := os.Getwd()
			Expect(err).ShouldNot(HaveOccurred(), "Failed to get the working directory")
			err = os.Chdir(filepath.Join(cwd + "../../../.."))
			Expect(err).ShouldNot(HaveOccurred())

			gd.Cli, err = gauntlet.NewGauntlet()
			Expect(err).ShouldNot(HaveOccurred())

			// terraNodeUrl, err := state.Env.Charts.Connections("localterra").LocalURLByPort("lcd", environment.HTTP)
			// TODO: this needs to be supported by the helm chart
			lcdUri := state.Env.URLs["localterra"][0]
			terraNodeUrl, err := url.Parse(lcdUri)
			Expect(err).ShouldNot(HaveOccurred())
			gd.Cli.NetworkConfig = integration_tests.GetDefaultGauntletConfig(terraNodeUrl)
			err = gd.Cli.WriteNetworkConfigMap(utils.Networks)
			Expect(err).ShouldNot(HaveOccurred(), "failed to write the .env file")
			gd.Cli.NetworkConfig["LINK"] = gd.LinkToken
		})
	})

	Describe("Run Gauntlet Commands", func() {
		It("should deploy ocr and accept a proposal", func() {
			// upload artifacts
			gd.Upload()

			gd.LinkToken = gd.DeployToken()
			gd.Cli.NetworkConfig["LINK"] = gd.LinkToken
			err := common.FundOracles(state.NodeKeysBundle, big.NewFloat(5e12))
			Expect(err).ShouldNot(HaveOccurred())

			// deploy access controllers
			gd.BillingAccessController = gd.DeployBillingAccessController()
			gd.RequesterAccessController = gd.DeployRequesterAccessController()

			// write the updated values for link and access controllers to the .env file
			err = gd.Cli.WriteNetworkConfigMap(utils.Networks)
			Expect(err).ShouldNot(HaveOccurred(), "Failed to write the updated .env file")

			// flags:deploy
			gd.Flags = gd.DeployFlags(gd.BillingAccessController, gd.RequesterAccessController)

			// deviation_flagging_validator:deploy
			gd.DeviationFlaggingValidator = gd.DeployDeviationFlaggingValidator(gd.Flags, 8000)

			// ocr2:deploy
			gd.OCR, gd.RddPath = gd.DeployOcr()

			// ocr2:set_billing
			gd.SetBilling(gd.OCR, gd.RddPath)

			// ocr2:begin_proposal
			gd.ProposalId = gd.BeginProposal(gd.OCR, gd.RddPath)

			// ocr2:propose_config
			gd.ProposeConfig(gd.OCR, gd.ProposalId, gd.RddPath)

			// ocr2:propose_offchain_config
			gd.OffchainProposalSecret = gd.ProposeOffchainConfig(gd.OCR, gd.ProposalId, gd.RddPath)

			// ocr2:finalize_proposal
			gd.ProposalDigest = gd.FinalizeProposal(gd.OCR, gd.ProposalId, gd.RddPath)

			// ocr2:accept_proposal
			gd.AcceptProposal(gd.OCR, gd.ProposalId, gd.ProposalDigest, gd.OffchainProposalSecret, gd.RddPath)

			// ocr2:inspect
			results := gd.OcrInspect(gd.OCR, gd.RddPath)
			Expect(len(results)).Should(Equal(28), "Did not find the expected number of results in the output")
			for _, v := range results {
				Expect(v.Pass).Should(Equal(true), fmt.Sprintf("%s expected %s but actually %s", v.Key, v.Expected, v.Actual))

			}
		})
	})

	AfterEach(func() {
		By("Tearing down the environment", func() {
			err := actions.TeardownSuite(state.Env, "logs", state.Nodes, nil, nil)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
