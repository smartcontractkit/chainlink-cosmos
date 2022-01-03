use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::Addr;
use cw_storage_plus::Item;

use owned::Auth;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct State {
    pub flagging_threshold: u32,
    pub flags: Addr,
}

pub const OWNER: Auth = Auth::new("owner");
pub const CONFIG: Item<State> = Item::new("config");
