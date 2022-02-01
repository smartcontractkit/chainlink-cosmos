package ocr2types

import (
	"github.com/smartcontractkit/terra.go/msg"
)

const (
	QueryLatestConfigDetails        = "latest_config_details"
	QueryTransmitters               = "transmitters"
	QueryLatestTransmissionDetails  = "latest_transmission_details"
	QueryLatestConfigDigestAndEpoch = "latest_config_digest_and_epoch"
	QueryDescription                = "description"
	QueryDecimals                   = "decimals"
	QueryLatestRoundData            = "latest_round_data"
	QueryLinkToken                  = "link_token"
	QueryBilling                    = "billing"
	QueryBillingAccessController    = "billing_access_controller"
	QueryRequesterAccessController  = "requester_access_controller"
	QueryLinkAvailableForPayment    = "link_available_for_payment"
)

type QueryLatestRoundDataResponse struct {
	QueryResult struct {
		Answer                string `json:"answer"`
		ObservationsTimestamp uint64 `json:"observations_timestamp"`
		RoundID               uint64 `json:"round_id"`
		TransmissionTimestamp uint64 `json:"transmission_timestamp"`
	} `json:"query_result"`
}

type QueryOwedPaymentMsg struct {
	OwedPayment QueryOwedPaymentTypeMsg `json:"owed_payment"`
}

type QueryOwedPaymentTypeMsg struct {
	Transmitter msg.AccAddress `json:"transmitter"`
}

type QueryRoundDataMsg struct {
	RoundData QueryRoundDataTypeMsg `json:"round_data"`
}

type QueryRoundDataTypeMsg struct {
	RoundID uint32 `json:"round_id"`
}

type OCRv2InstantiateMsg struct {
	BillingAccessController   string `json:"billing_access_controller"`
	RequesterAccessController string `json:"requester_access_controller"`
	LinkToken                 string `json:"link_token"`
	Decimals                  uint8  `json:"decimals"`
	Description               string `json:"description"`
	MinAnswer                 string `json:"min_answer"`
	MaxAnswer                 string `json:"max_answer"`
}

// ExecuteSetValidator execute set validator msg
type ExecuteSetValidator struct {
	SetValidator ExecuteSetValidatorConfig `json:"set_validator_config"`
}

// ExecuteSetValidatorConfig execute set validator msg
type ExecuteSetValidatorConfig struct {
	Config ExecuteSetValidatorConfigType `json:"config"`
}

// ExecuteSetValidatorConfigType execute set validator msg
type ExecuteSetValidatorConfigType struct {
	Address  string `json:"address"`
	GasLimit uint64 `json:"gas_limit"`
}

// ExecuteSetPayees set payees msg
type ExecuteSetPayees struct {
	SetPayees ExecuteSetPayeesConfig `json:"set_payees"`
}

// ExecuteSetPayeesConfig set payees msg
type ExecuteSetPayeesConfig struct {
	Payees [][]string `json:"payees"`
}

type ExecuteSetBillingMsg struct {
	SetBilling ExecuteSetBillingMsgType `json:"set_billing"`
}

type ExecuteSetBillingMsgType struct {
	Config ExecuteSetBillingConfigMsgType `json:"config"`
}

type ExecuteSetBillingConfigMsgType struct {
	BaseGas             uint64 `json:"base_gas"`
	TransmissionPayment uint64 `json:"transmission_payment_gjuels"`
	ObservationPayment  uint64 `json:"observation_payment_gjuels"`
	RecommendedGasPrice string `json:"recommended_gas_price_uluna"`
}

type ExecuteTransferOwnershipMsg struct {
	TransferOwnership ExecuteTransferOwnershipMsgType `json:"transfer_ownership"`
}

type ExecuteTransferOwnershipMsgType struct {
	To msg.AccAddress `json:"to"`
}

type ExecuteSetConfig struct {
	SetConfig SetConfigDetails `json:"set_config"`
}

type SetConfigDetails struct {
	Signers               [][]byte `json:"signers"`
	Transmitters          []string `json:"transmitters"`
	F                     uint8    `json:"f"`
	OnchainConfig         []byte   `json:"onchain_config"`
	OffchainConfigVersion uint64   `json:"offchain_config_version"`
	OffchainConfig        []byte   `json:"offchain_config"`
}
