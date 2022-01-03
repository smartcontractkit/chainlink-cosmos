use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::Addr;
use cw_storage_plus::{Item, Map};

use access_controller::AccessControllerContract;
use owned::Auth;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Config {
    pub raising_access_controller: AccessControllerContract,
    pub lowering_access_controller: AccessControllerContract,
}

pub const CONFIG: Item<Config> = Item::new("config");
pub const OWNER: Auth = Auth::new("owner");
pub const FLAGS: Map<&Addr, ()> = Map::new("flags");
