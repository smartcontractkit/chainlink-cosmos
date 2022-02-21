package migration_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/common"
	tc "github.com/smartcontractkit/chainlink-terra/tests/e2e/smoke/common"
	"github.com/smartcontractkit/integrations-framework/actions"
)

var _ = Describe("Terra OCRv2 @ocr-spec-migration", func() {
	var state *tc.OCRv2State
	var nodes = 5
	var rounds = 5
	var migrateToImage string
	var migrateToVersion string

	BeforeEach(func() {
		state = &tc.OCRv2State{}
		By("Deoloying the cluster", func() {
			migrateToImage = os.Getenv("CHAINLINK_IMAGE_TO")
			if migrateToImage == "" {
				Fail("Provide CHAINLINK_IMAGE_TO variable: an image on which we migrate")
			}
			migrateToVersion = os.Getenv("CHAINLINK_VERSION_TO")
			if migrateToVersion == "" {
				Fail("Provide CHAINLINK_VERSION_TO variable: a version on which we migrate")
			}
			state.DeployCluster(nodes, true)
			common.ImitateSource(state.MockServer, 1*time.Second, 2, 10)
		})
	})

	Describe("with Terra OCR2", func() {
		It("performs OCR2 round", func() {
			state.ValidateRoundsAfter(time.Now(), rounds)
			state.UpdateChainlinkVersion(migrateToImage, migrateToVersion)
			state.ValidateRoundsAfter(time.Now(), rounds)
		})
	})

	AfterEach(func() {
		By("Tearing down the environment", func() {
			err := actions.TeardownSuite(state.Env, nil, "logs", nil)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
