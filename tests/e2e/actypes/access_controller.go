package actypes

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type InstantiateMsg struct{}

type ExecuteAddAccessMsg struct {
	AddAccess ExecuteAddAccessTypeMsg `json:"add_access"`
}

type ExecuteAddAccessTypeMsg struct {
	Address sdk.AccAddress `json:"address"`
}

type ExecuteRemoveAccessMsg struct {
	RemoveAccess ExecuteRemoveAccessTypeMsg `json:"remove_access"`
}

type ExecuteRemoveAccessTypeMsg struct {
	Address sdk.AccAddress `json:"address"`
}

type QueryHasAccessMsg struct {
	HasAccess QueryHasAccessTypeMsg `json:"has_access"`
}

type QueryHasAccessTypeMsg struct {
	Address sdk.AccAddress `json:"address"`
}

type QueryHasAccessResponse struct {
	QueryResult bool `json:"query_result"`
}
