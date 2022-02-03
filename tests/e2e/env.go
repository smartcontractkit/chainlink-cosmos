package e2e

import "github.com/smartcontractkit/helmenv/environment"

// NewChainlinkTerraEnv returns a cluster config with LocalTerra node
func NewChainlinkTerraEnv(nodes int, stateful bool) *environment.Config {
	env := &environment.Config{
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
					"replicas": nodes,
					"chainlink": map[string]interface{}{
						"image": map[string]interface{}{
							"image":   "795953128386.dkr.ecr.us-west-2.amazonaws.com/chainlink",
							"version": "develop.latest",
						},
					},
					"env": map[string]interface{}{
						"EVM_ENABLED":                 "false",
						"EVM_RPC_ENABLED":             "false",
						"TERRA_ENABLED":               "true",
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
	if stateful {
		env.Charts["chainlink"].Values["db"] = map[string]interface{}{
			"stateful": true,
			"capacity": "2Gi",
		}
	}
	return env
}
