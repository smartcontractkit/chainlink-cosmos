use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {
    pub raising_access_controller: String,
    pub lowering_access_controller: String,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
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

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    /// Returns contract owner's address
    /// Response [`Addr`]
    Owner,
    Flag {
        subject: String,
    },
    Flags {
        subjects: Vec<String>,
    },
    RaisingAccessController,
}
