package smoke_test

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/common"
	tc "github.com/smartcontractkit/chainlink-terra/tests/e2e/smoke/common"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/utils"
	"github.com/smartcontractkit/helmenv/environment"
	"github.com/smartcontractkit/integrations-framework/actions"
	"github.com/smartcontractkit/integrations-framework/gauntlet"
)

var _ = Describe("Terra Gauntlet @gauntlet", func() {
	var (
		gd    *e2e.GauntletDeployer
		state *tc.OCRv2State
	)

	BeforeEach(func() {
		By("Deploying the environment", func() {
			gd = &e2e.GauntletDeployer{}
			state = &tc.OCRv2State{}
			state.DeployEnv(1, false)
			state.SetupClients()
			if state.Nets.Default.ContractsDeployed() {
				err := state.LoadContracts()
				Expect(err).ShouldNot(HaveOccurred())
			}

			state.OCConfig, state.NodeKeysBundle, state.Err = common.DefaultOffChainConfigParamsFromNodes(state.Nodes)
			Expect(state.Err).ShouldNot(HaveOccurred())

			cd := e2e.NewTerraContractDeployer(state.Nets.Default)

			linkToken, err := cd.DeployLinkTokenContract()
			Expect(err).ShouldNot(HaveOccurred(), "Failed to deploy link token")
			gd.LinkToken = linkToken.Address()

			err = common.FundOracles(state.Nets.Default, state.NodeKeysBundle, big.NewFloat(5e12))
			Expect(err).ShouldNot(HaveOccurred())
		})
		By("Setup Gauntlet", func() {
			cwd, err := os.Getwd()
			Expect(err).ShouldNot(HaveOccurred(), "Failed to get the working directory")
			err = os.Chdir(filepath.Join(cwd + "../../../.."))
			Expect(err).ShouldNot(HaveOccurred())

			gd.Cli, err = gauntlet.NewGauntlet()
			Expect(err).ShouldNot(HaveOccurred())

			terraNodeUrl, err := state.Env.Charts.Connections("localterra").LocalURLByPort("lcd", environment.HTTP)
			Expect(err).ShouldNot(HaveOccurred())
			gd.Cli.NetworkConfig = e2e.GetDefaultGauntletConfig(terraNodeUrl)
			err = gd.Cli.WriteNetworkConfigMap(utils.Networks)
			Expect(err).ShouldNot(HaveOccurred(), "failed to write the .env file")
			gd.Cli.NetworkConfig["LINK"] = gd.LinkToken
		})
	})

	Describe("Run Gauntlet Commands", func() {
		It("should deploy ocr and accept a proposal", func() {
			// upload artifacts
			gd.Upload()

			// deploy access controllers
			gd.DeployBillingAccessController()
			gd.DeployRequesterAccessController()

			// write the updated values for link and access controllers to the .env file
			err := gd.Cli.WriteNetworkConfigMap(utils.Networks)
			Expect(err).ShouldNot(HaveOccurred(), "Failed to write the updated .env file")

			// flags:deploy
			gd.DeployFlags()

			// deviation_flagging_validator:deploy
			gd.DeployDeviationFlaggingValidator()

			// ocr2:deploy
			gd.DeployOcr()

			// ocr2:set_billing
			gd.SetBilling()

			// ocr2:begin_proposal
			gd.BeginProposal()

			// ocr2:propose_config
			gd.ProposeConfig()

			// ocr2:propose_offchain_config
			gd.ProposeOffchainConfig()

			// ocr2:finalize_proposal
			gd.FinalizeProposal()

			// ocr2:accept_proposal
			gd.AcceptProposal()

			// ocr2:inspect
			results := gd.OcrInspect()

			Expect(len(results)).Should(Equal(12), "Did not find the expected number of results in the output")
			for k, v := range results {
				// skipping min/max answer because they do not get populated
				// skipping link available because we didn't transfer an link in this test
				if k == "Min Answer" || k == "Max Answer" || k == "LINK Available" {
					Expect(v.Pass).Should(Equal(false), fmt.Sprintf("%s is expected to fail", v.Key))
				} else {
					Expect(v.Pass).Should(Equal(true), fmt.Sprintf("%s expected %s but actually %s", v.Key, v.Expected, v.Actual))
				}
			}
		})
	})

	AfterEach(func() {
		By("Tearing down the environment", func() {
			err := actions.TeardownSuite(state.Env, nil, "logs", nil)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
