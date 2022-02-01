use cosmwasm_std::Addr;
use cw_storage_plus::Map;
use owned::Auth;

pub const OWNER: Auth = Auth::new("owner");

// TODO: could use SnapshotMap to store historic access data
pub const ACCESS: Map<&Addr, ()> = Map::new("access");
