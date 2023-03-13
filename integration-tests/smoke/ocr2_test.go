package smoke_test

import (
	"errors"
	"testing"

	"github.com/test-go/testify/require"
)

func TestOCRBasic(t *testing.T) {
	require.NoError(t, errors.New("implement me"))
}

// var _ = Describe("Terra OCRv2 @ocr2", func() {
// 	var state *tc.OCRv2State

// 	BeforeEach(func() {
// 		state = tc.NewOCRv2State(1, 5)
// 		By("Deploying the cluster", func() {
// 			state.DeployCluster(5, common.ChainBlockTime, false, utils.ContractsDir)
// 			state.SetAllAdapterResponsesToTheSameValue(2)
// 		})
// 	})

// 	Describe("with Terra OCR2", func() {
// 		It("performs OCR2 round", func() {
// 			state.ValidateAllRounds(time.Now(), tc.NewRoundCheckTimeout, 10, false)
// 		})
// 	})

// 	AfterEach(func() {
// 		By("Tearing down the environment", func() {
// 			err := actions.TeardownSuite(state.Env, "logs", state.Nodes, nil, nil)
// 			Expect(err).ShouldNot(HaveOccurred())
// 		})
// 	})
// })
