package monitoring

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// ChainReader is a subset of the pkg/terra/client.Reader interface
// that is used by this envelope source.
type ChainReader interface {
	TxsEvents(events []string, paginationParams *query.PageRequest) (*txtypes.GetTxsEventResponse, error)
	ContractStore(contractAddress sdk.AccAddress, queryMsg []byte) ([]byte, error)
}
