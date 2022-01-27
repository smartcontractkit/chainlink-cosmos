package validatortypes

type InstantiateMsg struct {
	FlaggingThreshold uint32 `json:"flagging_threshold"`
	Flags             string `json:"flags"`
}
