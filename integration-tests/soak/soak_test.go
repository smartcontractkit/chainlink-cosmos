package soak_test

// var _ = Describe("Terra OCRv2 soak test @ocr2-soak", func() {
// 	var state *tc.OCRv2State

// 	BeforeEach(func() {
// 		state = tc.NewOCRv2State(30, 5)
// 		By("Deploying the cluster", func() {
// 			state.DeployCluster(5, common.ChainBlockTimeSoak, false, utils.ContractsDir)
// 			state.SetAllAdapterResponsesToTheSameValue(2)
// 		})
// 	})

// 	Describe("with Terra OCR2", func() {
// 		It("performs OCR2 round", func() {
// 			state.ValidateAllRounds(time.Now(), tc.NewSoakRoundCheckTimeout, 300, false)
// 		})
// 	})

// 	AfterEach(func() {
// 		By("Tearing down the environment", func() {
// 			err := actions.TeardownSuite(state.Env, "logs", state.Nodes, nil, nil)
// 			Expect(err).ShouldNot(HaveOccurred())
// 		})
// 	})
// })
