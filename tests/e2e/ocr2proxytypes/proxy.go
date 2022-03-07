package ocr2proxytypes

type InstantiateMsg struct {
	ContractAddress string `json:"contract_address"`
}

type ProposeContractMsg struct {
	ContractAddress string `json:"propose_contract"`
}

type ConfirmContractMsg struct {
	ContractAddress string `json:"confirm_contract"`
}

type TransferOwnershipMsg struct {
	ToAddress string `json:"transfer_ownership"`
}

type ContractAddress struct {
	Address string `json:"address"`
}

type ToAddress struct {
	To string `json:"to"`
}
