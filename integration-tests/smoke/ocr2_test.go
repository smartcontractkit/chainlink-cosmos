package smoke_test

import (
	"time"

	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/common"
	"github.com/smartcontractkit/chainlink-cosmos/integration-tests/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/smartcontractkit/chainlink/integration-tests/actions"

	tc "github.com/smartcontractkit/chainlink-cosmos/integration-tests/smoke/common"
)

var _ = Describe("Terra OCRv2 @ocr2", func() {
	var state *tc.OCRv2State

	BeforeEach(func() {
		state = tc.NewOCRv2State(1, 5)
		By("Deploying the cluster", func() {
			state.DeployCluster(5, common.ChainBlockTime, false, utils.ContractsDir)
			state.SetAllAdapterResponsesToTheSameValue(2)
		})
	})

	Describe("with Terra OCR2", func() {
		It("performs OCR2 round", func() {
			state.ValidateAllRounds(time.Now(), tc.NewRoundCheckTimeout, 10, false)
		})
	})

	AfterEach(func() {
		By("Tearing down the environment", func() {
			err := actions.TeardownSuite(state.Env, "logs", state.Nodes, nil, nil)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
