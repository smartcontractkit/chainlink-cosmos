package migration_test

import (
	"os"
	"time"

	"github.com/smartcontractkit/chainlink-terra/tests/e2e/common"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tc "github.com/smartcontractkit/chainlink-terra/tests/e2e/smoke/common"
	"github.com/smartcontractkit/chainlink-testing-framework/actions"
)

var _ = Describe("Terra OCRv2 @ocr-spec-migration", func() {
	var state *tc.OCRv2State
	var nodes = 5
	var rounds = 5
	var migrateToImage string
	var migrateToVersion string

	BeforeEach(func() {
		state = tc.NewOCRv2State(1, nodes)
		By("Deploying the cluster", func() {
			migrateToImage = os.Getenv("CHAINLINK_IMAGE_TO")
			if migrateToImage == "" {
				Fail("Provide CHAINLINK_IMAGE_TO variable: an image on which we migrate")
			}
			migrateToVersion = os.Getenv("CHAINLINK_VERSION_TO")
			if migrateToVersion == "" {
				Fail("Provide CHAINLINK_VERSION_TO variable: a version on which we migrate")
			}
			state.DeployCluster(nodes, common.ChainBlockTime, true, utils.ContractsDir)
			state.SetAllAdapterResponsesToTheSameValue(2)
		})
	})

	Describe("with Terra OCR2", func() {
		It("performs OCR2 round", func() {
			state.ValidateAllRounds(time.Now(), tc.NewRoundCheckTimeout, rounds, false)
			state.UpdateChainlinkVersion(migrateToImage, migrateToVersion)
			state.ValidateAllRounds(time.Now(), tc.NewRoundCheckTimeout, rounds, false)
		})
	})

	AfterEach(func() {
		By("Tearing down the environment", func() {
			err := actions.TeardownSuite(state.Env, "logs", state.Nodes, nil, nil)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
