package soak_test

import (
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/common"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tc "github.com/smartcontractkit/chainlink-terra/tests/e2e/smoke/common"
	"github.com/smartcontractkit/integrations-framework/actions"
)

var _ = Describe("Terra OCRv2 soak test @ocr2-soak", func() {
	var state *tc.OCRv2State

	BeforeEach(func() {
		state = tc.NewOCRv2State(10)
		By("Deploying the cluster", func() {
			state.DeployCluster(5, common.ChainBlockTimeSoak, false)
			state.SetAllAdapterResponsesToTheSameValue(2)
		})
	})

	Describe("with Terra OCR2", func() {
		It("performs OCR2 round", func() {
			state.ValidateAllRounds(time.Now(), 50, false)
		})
	})

	AfterEach(func() {
		By("Tearing down the environment", func() {
			err := actions.TeardownSuite(state.Env, nil, "logs", nil)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
