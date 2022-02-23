package e2e

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/smartcontractkit/terra.go/key"
	"github.com/smartcontractkit/terra.go/msg"
)

type RddConfig struct {
	Apis       map[string]interface{} `json:"apis"`
	Contracts  map[string]interface{} `json:"contracts"`
	Flags      map[string]interface{} `json:"flags"`
	Network    map[string]interface{} `json:"network"`
	Operators  map[string]interface{} `json:"operators"`
	Proxies    map[string]interface{} `json:"proxies"`
	Validators map[string]interface{} `json:"validators"`
}

// WriteRdd writes the rdd data to a file
func WriteRdd(rdd *RddConfig, file string) error {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	j, err := json.Marshal(rdd)
	if err != nil {
		return err
	}
	log.Info().Str("Out", string(j)).Msg("The stuff we are writing")

	_, err = f.Write(j)

	return err
}

// NewChainlinkTerraEnv returns a cluster config with LocalTerra node
func NewRddContract(contractId string) *RddConfig {
	rdd := &RddConfig{
		Apis: map[string]interface{}{},
		Contracts: map[string]interface{}{
			contractId: map[string]interface{}{
				"billing": map[string]interface{}{
					"observationPaymentGjuels":  "1",
					"recommendedGasPriceMicro":  "1.1",
					"transmissionPaymentGjuels": "1",
				},
				"config": map[string]interface{}{
					"deltaGrace":                             "1s",
					"deltaProgress":                          "12s",
					"deltaResend":                            "30s",
					"deltaRound":                             "10s",
					"deltaStage":                             "14s",
					"f":                                      1,
					"maxDurationObservation":                 "1s",
					"maxDurationQuery":                       "0s",
					"maxDurationReport":                      "5s",
					"maxDurationShouldAcceptFinalizedReport": "1s",
					"maxDurationShouldTransmitAcceptedReport": "1s",
					"rMax": 6,
					"reportingPluginConfig": map[string]interface{}{
						"alphaAcceptInfinite": false,
						"alphaAcceptPpb":      "3000000",
						"alphaReportInfinite": false,
						"alphaReportPpb":      "3000000",
						"deltaC":              "50s",
					},
					"s": []int{
						1,
						1,
						2,
						2,
					},
				},
				"contractVersion": 6,
				"decimals":        8,
				"docsHidden":      true,
				"externalAdapterRequestParams": map[string]interface{}{
					"from": "ETH",
					"to":   "USD",
				},
				"marketing": map[string]interface{}{
					"decimalPlaces":       2,
					"formatDecimalPlaces": 0,
					"history":             true,
					"pair": []string{
						"ETH",
						"USD",
					},
					"path": "eth-usd-ocr2",
				},
				"maxSubmissionValue": "99999999999999999999999999999",
				"minSubmissionValue": "0",
				"name":               "ETH / USD",
				"oracles": []interface{}{
					newOracle("node-1"),
					newOracle("node-2"),
					newOracle("node-3"),
					newOracle("node-4"),
				},
				"status": "live",
				"type":   "numerical_median_feed",
			},
		},
		Flags:   map[string]interface{}{},
		Network: map[string]interface{}{},
		Operators: map[string]interface{}{
			"node-1": newOperator("node-1", 1),
			"node-2": newOperator("node-2", 2),
			"node-3": newOperator("node-3", 3),
			"node-4": newOperator("node-4", 4),
		},
		Proxies:    map[string]interface{}{},
		Validators: map[string]interface{}{},
	}
	return rdd
}

func newOperator(nodeName string, index int) map[string]interface{} {
	// create a public key node address
	mnemonic, _ := key.CreateMnemonic()
	privKeyBz, _ := key.DerivePrivKeyBz(mnemonic, key.CreateHDPath(0, 0))
	privKey, _ := key.PrivKeyGen(privKeyBz)
	addr := msg.AccAddress(privKey.PubKey().Address())

	return map[string]interface{}{
		"displayName":  nodeName,
		"adminAddress": "terra1mskaupg53dc8jh50nstcjmctm4sud9fc2t8rjn",
		"csaKeys": []interface{}{
			map[string]interface{}{
				"nodeAddress": "terra1mskaupg53dc8jh50nstcjmctm4sud9fc2t8rjn",
				"nodeName":    "node 1",
				"publicKey":   fmt.Sprintf("c880f65f9e2118063c1e61b5f54c84c80651f2b8a367f46d3dbfbad4966c7f8%v", index),
			},
		},
		"ocr2ConfigPublicKey": []interface{}{
			fmt.Sprintf("ocr2cfg_terra_b90e50daf82024624549e7708199dd05b6de8e10d6df62cd27581c65e5096b2%v", index),
		},
		"ocr2OffchainPublicKey": []interface{}{
			fmt.Sprintf("ocr2off_terra_3bdd39af448a824cb6042b981274baf26f7501f2918ae825afc51a2442ef699%v", index),
		},
		"ocr2OnchainPublicKey": []interface{}{
			fmt.Sprintf("ocr2on_terra_9c41de50e875fbca65643ffe60f90e84f1b9b4871092b7c3cbf4eff4b07e454%v", index),
		},
		"ocrNodeAddress": []interface{}{
			addr,
		},
		"peerId": []interface{}{
			fmt.Sprintf("12D3KooWHzGXm2NSRgYcn6B3szqfEr486kq2ipAEPXdqmE6nE2a%v", index),
		},
		"status": "active",
	}
}

func newOracle(nodeName string) map[string]interface{} {
	return map[string]interface{}{
		"api": []string{
			nodeName,
		},
		"operator": nodeName,
	}
}
