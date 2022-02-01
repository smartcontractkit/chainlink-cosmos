use crate::state::{bignum, Billing, Validator};
use cosmwasm_std::{Addr, Binary, Uint128};
use cw20::Cw20ReceiveMsg;
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct InstantiateMsg {
    /// LINK token contract address
    pub link_token: String,
    /// RequestNewRound access controller address
    pub requester_access_controller: String,
    /// Billing access controller address
    pub billing_access_controller: String,

    #[serde(with = "bignum")]
    #[schemars(with = "String")]
    pub min_answer: i128,
    #[serde(with = "bignum")]
    #[schemars(with = "String")]
    pub max_answer: i128,

    pub decimals: u8,
    pub description: String,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    SetConfig {
        signers: Vec<Binary>,
        transmitters: Vec<String>,
        f: u8,
        onchain_config: Binary,
        offchain_config_version: u64,
        offchain_config: Binary,
    },
    TransferOwnership {
        to: String,
    },
    AcceptOwnership,

    Transmit {
        report_context: Binary,
        report: Binary,

        // TODO: use signatures: Vec<[u8; 32+64]>, when it becomes possible
        // https://github.com/GREsau/schemars/issues/111
        signatures: Vec<Binary>,
    },

    RequestNewRound,

    SetBilling {
        config: Billing,
    },

    SetValidatorConfig {
        config: Option<Validator>,
    },

    SetBillingAccessController {
        access_controller: String,
    },
    SetRequesterAccessController {
        access_controller: String,
    },

    WithdrawPayment {
        transmitter: String,
    },
    WithdrawFunds {
        recipient: String,
        amount: Uint128,
    },

    SetLinkToken {
        link_token: String,
        recipient: String,
    },

    /// Handler for LINK token Receive message
    Receive(Cw20ReceiveMsg),

    SetPayees {
        payees: Vec<(String, String)>,
    },
    TransferPayeeship {
        transmitter: String,
        proposed: String,
    },
    AcceptPayeeship {
        transmitter: String,
    },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    // BASE:
    LatestConfigDetails,
    Transmitters,
    LatestTransmissionDetails,
    LatestConfigDigestAndEpoch,

    Description,
    Decimals,
    RoundData { round_id: u32 },
    LatestRoundData,

    LinkToken,
    Billing,
    BillingAccessController,
    RequesterAccessController,
    OwedPayment { transmitter: String },
    LinkAvailableForPayment,
    OracleObservationCount { transmitter: String },
    Version,
    Owner,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct LatestConfigDetailsResponse {
    pub config_count: u32,
    pub block_number: u64,
    pub config_digest: [u8; 32],
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct TransmittersResponse {
    pub addresses: Vec<Addr>,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct LatestTransmissionDetailsResponse {
    pub latest_config_digest: [u8; 32],
    pub epoch: u32,
    pub round: u8,
    #[serde(with = "bignum")]
    #[schemars(with = "String")]
    pub latest_answer: i128,
    pub latest_timestamp: u32,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct LatestConfigDigestAndEpochResponse {
    pub scan_logs: bool,
    pub config_digest: [u8; 32],
    pub epoch: u32,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct LinkAvailableForPaymentResponse {
    #[serde(with = "bignum")]
    #[schemars(with = "String")]
    pub amount: i128,
}
