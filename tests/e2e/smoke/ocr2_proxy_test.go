package smoke_test

import (
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/common"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e/utils"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/smartcontractkit/chainlink-terra/tests/e2e"
	tc "github.com/smartcontractkit/chainlink-terra/tests/e2e/smoke/common"
	"github.com/smartcontractkit/integrations-framework/actions"
)

var _ = Describe("Terra OCRv2 Proxy @ocr_proxy", func() {
	var state *tc.OCRv2State

	BeforeEach(func() {
		state = tc.NewOCRv2State(1, 5)
		By("Deploying the cluster", func() {
			state.DeployCluster(5, common.ChainBlockTime, false, utils.ContractsDir)
			state.SetAllAdapterResponsesToTheSameValue(2)
		})
	})

	Describe("with Terra OCR2 Proxy", func() {
		It("performs OCR2 round through proxy", func() {
			expectedDecimals := 8
			expectedDescription := "ETH/USD"

			cd := e2e.NewTerraContractDeployer(state.Nets.Default)

			// deploy the proxy pointing at the ocr2 address
			ocrProxy, err := cd.DeployOCRv2Proxy(state.Contracts[0].OCR2.Address(), utils.ContractsDir)
			Expect(err).ShouldNot(HaveOccurred())

			// latestRoundData
			state.ValidateAllRounds(time.Now(), tc.NewRoundCheckTimeout, 10, true)

			// decimals
			dec, err := ocrProxy.GetDecimals()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(dec).Should(Equal(expectedDecimals))

			// description
			desc, err := ocrProxy.GetDescription()
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
