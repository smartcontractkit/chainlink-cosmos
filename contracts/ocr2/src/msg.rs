use crate::state::{bignum, Billing, Proposal, ProposalId, Round, Validator};
use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Addr, Binary, Uint128};
use cw20::Cw20ReceiveMsg;

#[cw_serde]
#[derive(Eq)]
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

#[cw_serde]
pub enum ExecuteMsg {
    BeginProposal,
    ClearProposal {
        id: ProposalId,
    },
    FinalizeProposal {
        id: ProposalId,
    },
    AcceptProposal {
        id: ProposalId,
        digest: Binary,
    },
    ProposeConfig {
        id: ProposalId,
        signers: Vec<Binary>,
        transmitters: Vec<String>,
        payees: Vec<String>,
        f: u8,
        onchain_config: Binary,
    },
    ProposeOffchainConfig {
        id: ProposalId,
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

    TransferPayeeship {
        transmitter: String,
        proposed: String,
    },
    AcceptPayeeship {
        transmitter: String,
    },
}

#[cw_serde]
#[derive(Eq, QueryResponses)]
pub enum QueryMsg {
    // BASE:
    #[returns(LatestConfigDetailsResponse)]
    LatestConfigDetails,
    #[returns(TransmittersResponse)]
    Transmitters,
    #[returns(LatestTransmissionDetailsResponse)]
    LatestTransmissionDetails,
    #[returns(LatestConfigDigestAndEpochResponse)]
    LatestConfigDigestAndEpoch,
    #[returns(String)]
    Description,
    #[returns(u8)]
    Decimals,
    #[returns(Round)]
    RoundData { round_id: u32 },
    #[returns(Round)]
    LatestRoundData,
    #[returns(Addr)]
    LinkToken,
    #[returns(Billing)]
    Billing,
    #[returns(Addr)]
    BillingAccessController,
    #[returns(Addr)]
    RequesterAccessController,
    #[returns(Uint128)]
    OwedPayment { transmitter: String },
    #[returns(LinkAvailableForPaymentResponse)]
    LinkAvailableForPayment,
    #[returns(u32)]
    OracleObservationCount { transmitter: String },
    #[returns(Proposal)]
    Proposal { id: ProposalId },
    #[returns(str)]
    Version,
    #[returns(Addr)]
    Owner,
}

#[cw_serde]
#[derive(Eq)]
pub struct LatestConfigDetailsResponse {
    pub config_count: u32,
    pub block_number: u64,
    pub config_digest: [u8; 32],
}

#[cw_serde]
#[derive(Eq)]
pub struct TransmittersResponse {
    pub addresses: Vec<Addr>,
}

#[cw_serde]
#[derive(Eq)]
pub struct LatestTransmissionDetailsResponse {
    pub latest_config_digest: [u8; 32],
    pub epoch: u32,
    pub round: u8,
    #[serde(with = "bignum")]
    #[schemars(with = "String")]
    pub latest_answer: i128,
    pub latest_timestamp: u32,
}

#[cw_serde]
#[derive(Eq)]
pub struct LatestConfigDigestAndEpochResponse {
    pub scan_logs: bool,
    pub config_digest: [u8; 32],
    pub epoch: u32,
}

#[cw_serde]
#[derive(Eq)]
pub struct LinkAvailableForPaymentResponse {
    #[serde(with = "bignum")]
    #[schemars(with = "String")]
    pub amount: i128,
}
