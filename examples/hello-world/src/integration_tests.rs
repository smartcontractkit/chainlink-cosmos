#![cfg(test)]
#![cfg(not(tarpaulin_include))]
use crate::contract::{execute, instantiate, query};
use crate::msg::{ExecuteMsg, InstantiateMsg, QueryMsg};
use cosmwasm_std::{Addr, Empty};
use cw_multi_test::{App, AppBuilder, Contract, ContractWrapper, Executor};

fn mock_app() -> App {
    AppBuilder::new().build()
}

pub fn contract_hello_world() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(execute, instantiate, query);
    Box::new(contract)
}

struct Env {
    router: App,
    owner: Addr,
    hello_world_addr: Addr,
}

fn setup() -> Env {
    let mut router = mock_app();

    let owner = Addr::unchecked("owner");

    let hello_world_id = router.store_code(contract_hello_world());

    let hello_world_addr = router
        .instantiate_contract(
            hello_world_id,
            owner.clone(),
            &InstantiateMsg {
                feed: "chainlink_feed".to_string(),
                decimals: 18,
            },
            &[],
            "hello_world",
            None,
        )
        .unwrap();

    Env {
        router,
        owner,
        hello_world_addr,
    }
}

#[test]
fn proper_initialization() {
    setup();
}

#[test]
fn it_works() {
    let mut env = setup();

    // execute run()
    env.router
        .execute_contract(
            env.owner.clone(),
            env.hello_world_addr.clone(),
            &ExecuteMsg::Run {},
            &[],
        )
        .unwrap();

    // query price
    let price: bool = env
        .router
        .wrap()
        .query_wasm_smart(&env.hello_world_addr, &QueryMsg::Price {})
        .unwrap();
}
