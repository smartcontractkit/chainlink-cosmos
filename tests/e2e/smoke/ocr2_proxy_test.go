package smoke_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/common"
	tc "github.com/smartcontractkit/chainlink-terra/tests/e2e/smoke/common"
	"github.com/smartcontractkit/integrations-framework/actions"
)

var _ = Describe("Terra OCRv2 Proxy @ocr_proxy", func() {
	var state *tc.OCRv2State

	BeforeEach(func() {
		state = &tc.OCRv2State{}
		By("Deoloying the cluster", func() {
			state.DeployCluster(5, false)
			common.ImitateSource(state.MockServer, 1*time.Second, 2, 10)
		})
	})

	Describe("with Terra OCR2", func() {
		It("performs OCR2 round through proxy", func() {
			expectedDecimals := 8
			expectedDescription := "ETH/USD"

			cd := e2e.NewTerraContractDeployer(state.Nets.Default)

			// deploy the proxy pointing at the ocr2 address
			state.OCR2Proxy, state.Err = cd.DeployOCRv2Proxy(state.OCR2.Address())
			Expect(state.Err).ShouldNot(HaveOccurred())

			// latestRoundData
			state.ValidateRoundsAfter(time.Now(), 10, true)

			// decimals
			dec, err := state.OCR2Proxy.GetDecimals()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(dec).Should(Equal(expectedDecimals))

			// description
			desc, err := state.OCR2Proxy.GetDescription()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(desc).Should(Equal(expectedDescription))
		})
	})

	AfterEach(func() {
		By("Tearing down the environment", func() {
			err := actions.TeardownSuite(state.Env, nil, "logs", nil)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
