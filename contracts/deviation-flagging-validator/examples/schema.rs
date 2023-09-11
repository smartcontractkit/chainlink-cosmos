use std::env::current_dir;
use std::fs::{create_dir_all, remove_dir_all, rename};

use cosmwasm_schema::{export_schema, remove_schemas, schema_for, write_api};

use deviation_flagging_validator::msg::{ExecuteMsg, InstantiateMsg, QueryMsg};
use deviation_flagging_validator::state::State;

fn main() {
    // clean directory
    let mut out_dir = current_dir().unwrap();
    out_dir.push("schema");
    remove_dir_all(&out_dir).unwrap();
    create_dir_all(&out_dir).unwrap();

    write_api! {
        instantiate: InstantiateMsg,
        execute: ExecuteMsg,
        query: QueryMsg,
    }
}
