package e2e

import "github.com/smartcontractkit/helmenv/environment"

// NewChainlinkTerraEnv returns a cluster config with LocalTerra node
func NewChainlinkTerraEnv() *environment.Config {
	return &environment.Config{
		NamespacePrefix: "chainlink-terra",
		Charts: environment.Charts{
			"localterra": {
				Index: 1,
			},
			"mockserver-config": {
				Index: 2,
			},
			"mockserver": {
				Index: 3,
			},
			"chainlink": {
				Index: 4,
				Values: map[string]interface{}{
					"replicas": 5,
					"chainlink": map[string]interface{}{
						"image": map[string]interface{}{
							"image":   "public.ecr.aws/chainlink/chainlink",
							"version": "develop.b9faa7983e88dbb4c75d5e75be0d11733f5d50d0",
						},
					},
					"env": map[string]interface{}{
						"NODE_TYPE":     "terra",
						"RELAY_NAME":    "terra",
						"RELAY_CHAINID": "terra",

						"eth_disabled":                "true",
						"CHAINLINK_DEV":               "true",
						"USE_LEGACY_ETH_ENV_VARS":     "false",
						"FEATURE_OFFCHAIN_REPORTING2": "true",
						"feature_external_initiators": "true",
						"P2P_NETWORKING_STACK":        "V2",
						"P2PV2_LISTEN_ADDRESSES":      "0.0.0.0:6690",
						"P2PV2_DELTA_DIAL":            "5s",
						"P2PV2_DELTA_RECONCILE":       "5s",
						"p2p_listen_port":             "0",
					},
				},
			},
		},
	}
}
