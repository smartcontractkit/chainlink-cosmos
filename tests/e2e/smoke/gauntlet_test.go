package smoke_test

import (
	"math/big"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/common"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/utils"
	"github.com/smartcontractkit/helmenv/environment"
	"github.com/smartcontractkit/helmenv/tools"
	"github.com/smartcontractkit/integrations-framework/actions"
	"github.com/smartcontractkit/integrations-framework/client"
	"github.com/smartcontractkit/integrations-framework/gauntlet"
)

var _ = Describe("Terra Gauntlet @gauntlet", func() {
	var (
		e              *environment.Environment
		g              *gauntlet.Gauntlet
		nodes          []client.Chainlink
		nets           *client.Networks
		nkb            []common.NodeKeysBundle
		err            error
		networkDirPath string
	)

	terraCommandError := "Terra Command execution error"

	BeforeEach(func() {
		By("Deploying the environment", func() {
			e, err = environment.DeployOrLoadEnvironment(
				e2e.NewChainlinkTerraEnv(1, false),
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
			nodes, err = client.ConnectChainlinkNodes(e)
			Expect(err).ShouldNot(HaveOccurred())
		})
		By("Funding Wallets", func() {
			_, nkb, err = common.DefaultOffChainConfigParamsFromNodes(nodes)
			Expect(err).ShouldNot(HaveOccurred())

			err = common.FundOracles(nets.Default, nkb, big.NewFloat(5e12))
			Expect(err).ShouldNot(HaveOccurred())
		})
		By("Setup Gauntlet", func() {
			networkDirPath = filepath.Join(utils.ProjectRoot, "./packages-ts/gauntlet-terra-contracts/networks")
			cwd, _ := os.Getwd()
			err = os.Chdir(filepath.Join(cwd + "../../../.."))
			Expect(err).ShouldNot(HaveOccurred())

			g, err = gauntlet.NewGauntlet()
			Expect(err).ShouldNot(HaveOccurred())

			terraNodeUrl, err := e.Charts.Connections("localterra").LocalURLByPort("lcd", environment.HTTP)
			Expect(err).ShouldNot(HaveOccurred())
			g.NetworkConfig = common.GetDefaultGauntletConfig(terraNodeUrl)
			err = g.WriteNetworkConfigMap(networkDirPath)
			Expect(err).ShouldNot(HaveOccurred(), "failed to write the .env file")
		})
	})

	Describe("Run Gauntlet Commands", func() {
		It("should upload the contracts", func() {
			_, err = g.ExecCommandWithRetries([]string{
				"upload",
				g.Flag("version", "local"),
			}, []string{
				"Error deploying",
				terraCommandError,
			}, 10)
			Expect(err).ShouldNot(HaveOccurred(), "Failed to upload contracts")
		})
	})

	AfterEach(func() {
		By("Tearing down the environment", func() {
			err = actions.TeardownSuite(e, nil, "logs", nil)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
