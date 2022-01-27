package flagstypes

type InstantiateMsg struct {
	LoweringAccessController string `json:"lowering_access_controller"`
	RaisingAccessController  string `json:"raising_access_controller"`
}
