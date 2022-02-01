package e2e

import "github.com/smartcontractkit/helmenv/environment"

// NewChainlinkTerraEnv returns a cluster config with LocalTerra node
func NewChainlinkTerraEnv(nodeCount int) *environment.Config {
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
					"replicas": nodeCount,
					"chainlink": map[string]interface{}{
						"image": map[string]interface{}{
							"image":   "public.ecr.aws/z0b1w9r9/chainlink",
							"version": "develop.3abacbfc0761b4f6cf4d4d897bb4f94d05a4f793",
						},
					},
					"env": map[string]interface{}{
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
