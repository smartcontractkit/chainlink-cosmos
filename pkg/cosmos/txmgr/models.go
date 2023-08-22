package txmgr

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	txmgrtypes "github.com/smartcontractkit/chainlink-relay/pkg/txmgr/types"
)

// Type aliases for Cosmos
type (
	TxRequest = txmgrtypes.TxRequest[sdk.AccAddress, common.Hash]
)
