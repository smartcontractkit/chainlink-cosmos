use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::Addr;
use cw_storage_plus::Item;

use chainlink_cosmos::state::Round;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
pub struct Config {
    pub feed: Addr,
    pub decimals: u8,
}

pub const CONFIG: Item<Config> = Item::new("config");

pub const PRICE: Item<Round> = Item::new("price");
