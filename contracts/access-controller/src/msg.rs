use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::Addr;
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct InstantiateMsg {}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    AddAccess { address: String },
    RemoveAccess { address: String },
    TransferOwnership { to: String },
    AcceptOwnership,
}

#[cw_serde]
#[derive(QueryResponses, Eq)]
pub enum QueryMsg {
    #[returns(bool)]
    HasAccess { address: String },
    #[returns(Addr)]
    Owner {},
}
