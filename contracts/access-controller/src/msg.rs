use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::Addr;

#[cw_serde]
#[derive(Eq)]
#[serde(rename_all = "snake_case")]
pub struct InstantiateMsg {}

#[cw_serde]
#[derive(Eq)]
pub enum ExecuteMsg {
    AddAccess { address: String },
    RemoveAccess { address: String },
    TransferOwnership { to: String },
    AcceptOwnership,
}

#[cw_serde]
#[derive(Eq, QueryResponses)]
pub enum QueryMsg {
    #[returns(bool)]
    HasAccess { address: String },
    #[returns(Addr)]
    Owner,
}
