package common

type OCR2Config struct {
	ProposalId            string          `json:"proposalId"`
	F                     int             `json:"f"`
	Signers               []string        `json:"signers"`
	Transmitters          []string        `json:"transmitters"`
	Payees                []string        `json:"payees"`
	OnchainConfig         string          `json:"onchainConfig"`
	OffchainConfig        *OffchainConfig `json:"offchainConfig"`
	OffchainConfigVersion int             `json:"offchainConfigVersion"`
	Secret                string          `json:"secret"`
}

type OffchainConfig struct {
	DeltaProgressNanoseconds                           int64                  `json:"deltaProgressNanoseconds"`
	DeltaResendNanoseconds                             int64                  `json:"deltaResendNanoseconds"`
	DeltaRoundNanoseconds                              int64                  `json:"deltaRoundNanoseconds"`
	DeltaGraceNanoseconds                              int                    `json:"deltaGraceNanoseconds"`
	DeltaStageNanoseconds                              int64                  `json:"deltaStageNanoseconds"`
	RMax                                               int                    `json:"rMax"`
	S                                                  []int                  `json:"s"`
	OffchainPublicKeys                                 []string               `json:"offchainPublicKeys"`
	PeerIds                                            []string               `json:"peerIds"`
	ReportingPluginConfig                              *ReportingPluginConfig `json:"reportingPluginConfig"`
	MaxDurationQueryNanoseconds                        int                    `json:"maxDurationQueryNanoseconds"`
	MaxDurationObservationNanoseconds                  int                    `json:"maxDurationObservationNanoseconds"`
	MaxDurationReportNanoseconds                       int                    `json:"maxDurationReportNanoseconds"`
	MaxDurationShouldAcceptFinalizedReportNanoseconds  int                    `json:"maxDurationShouldAcceptFinalizedReportNanoseconds"`
	MaxDurationShouldTransmitAcceptedReportNanoseconds int                    `json:"maxDurationShouldTransmitAcceptedReportNanoseconds"`
	ConfigPublicKeys                                   []string               `json:"configPublicKeys"`
}

type ReportingPluginConfig struct {
	AlphaReportInfinite bool `json:"alphaReportInfinite"`
	AlphaReportPpb      int  `json:"alphaReportPpb"`
	AlphaAcceptInfinite bool `json:"alphaAcceptInfinite"`
	AlphaAcceptPpb      int  `json:"alphaAcceptPpb"`
	DeltaCNanoseconds   int  `json:"deltaCNanoseconds"`
}

var TestOCR2Config = OCR2Config{
	// ProposalId:       proposalId, // user defined
	F: 1,
	// Signers:       onChainKeys, // user defined
	// Transmitters:  txKeys, // user defined
	OnchainConfig: "",
	OffchainConfig: &OffchainConfig{
		DeltaProgressNanoseconds: 8000000000,
		DeltaResendNanoseconds:   30000000000,
		DeltaRoundNanoseconds:    3000000000,
		DeltaGraceNanoseconds:    1000000000,
		DeltaStageNanoseconds:    20000000000,
		RMax:                     5,
		S:                        []int{1, 2},
		// OffchainPublicKeys:       offChainKeys, // user defined
		// PeerIds:                  peerIds, // user defined
		ReportingPluginConfig: &ReportingPluginConfig{
			AlphaReportInfinite: false,
			AlphaReportPpb:      0,
			AlphaAcceptInfinite: false,
			AlphaAcceptPpb:      0,
			DeltaCNanoseconds:   1000000000,
		},
		MaxDurationQueryNanoseconds:                        2000000000,
		MaxDurationObservationNanoseconds:                  1000000000,
		MaxDurationReportNanoseconds:                       2000000000,
		MaxDurationShouldAcceptFinalizedReportNanoseconds:  2000000000,
		MaxDurationShouldTransmitAcceptedReportNanoseconds: 2000000000,
		// ConfigPublicKeys:                                   cfgKeys, // user defined
	},
	OffchainConfigVersion: 2,
	Secret:                "awe accuse polygon tonic depart acuity onyx inform bound gilbert expire",
}

var TestOnKeys = []string{
	"0x04cc1bfa99e282e434aef2815ca17337a923cd2c61cf0c7de5b326d7a8603730",
	"0x04cc1bfa99e282e434aef2815ca17337a923cd2c61cf0c7de5b326d7a8603731",
	"0x04cc1bfa99e282e434aef2815ca17337a923cd2c61cf0c7de5b326d7a8603732",
	"0x04cc1bfa99e282e434aef2815ca17337a923cd2c61cf0c7de5b326d7a8603733",
}

var TestTxKeys = []string{
	"wasm1dd66lfqdcr9frs8vta7lwp9w565z6l7p2h6tmf",
	"wasm15nl46kf8aldtwefw6wee38flnhyealel5uwkcg",
	"wasm15xj6lehvtwgy0783znyj022twv784xfleacpn9",
	"wasm14x4qe4qakyd9xsq2r8xg2ld6s2q0h4zyqrxn9d",
}

var TestOffKeys = []string{
	"af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852090",
	"af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852091",
	"af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852092",
	"af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852093",
}

var TestCfgKeys = []string{
	"af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852094",
	"af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852095",
	"af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852096",
	"af400004fa5d02cd5170b5261032e71f2847ead36159cf8dee68affc3c852097",
}
