use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::Addr;
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[cw_serde]
#[derive(Eq)]
#[serde(rename_all = "snake_case")]

pub struct InstantiateMsg {
    pub raising_access_controller: String,
    pub lowering_access_controller: String,
}

#[cw_serde]
#[derive(Eq)]
pub enum ExecuteMsg {
    /// Initiate contract ownership transfer to another address.
    /// Can be used only by owner
    TransferOwnership {
        /// Address to transfer ownership to
        to: String,
    },
    /// Finish contract ownership transfer. Can be used only by pending owner
    AcceptOwnership,
    RaiseFlag {
        subject: String,
    },
    RaiseFlags {
        subjects: Vec<String>,
    },
    LowerFlags {
        subjects: Vec<String>,
    },
    SetRaisingAccessController {
        rac_address: String,
    },
}

#[cw_serde]
#[derive(Eq, QueryResponses)]
pub enum QueryMsg {
    #[returns(Addr)]
    Owner,
    #[returns(bool)]
    Flag { subject: String },
    #[returns(Vec<bool>)]
    Flags { subjects: Vec<String> },
    #[returns(Addr)]
    RaisingAccessController,
}
