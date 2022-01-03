#![cfg(test)]
#![cfg(not(tarpaulin_include))]
use cosmwasm_std::{
    to_binary, Addr, Binary, Deps, DepsMut, Empty, Env, MessageInfo, Response, StdResult,
};
use cw_multi_test::{App, AppBuilder, Contract, ContractWrapper, Executor};

use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum FooQueryMsg {
    Foo,
}

mod foo {
    pub use super::FooQueryMsg;
    crate::contract!(super::FooQueryMsg);
}

use foo::msg::{ExecuteMsg, InstantiateMsg};
use foo::{execute, instantiate, query};

fn mock_app() -> App {
    AppBuilder::new().build()
}

pub fn contract_proxy() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(execute, instantiate, query);
    Box::new(contract)
}

pub fn contract_foo() -> Box<dyn Contract<Empty>> {
    pub fn execute(
        _deps: DepsMut,
        _env: Env,
        _info: MessageInfo,
        _msg: ExecuteMsg,
    ) -> StdResult<Response> {
        unreachable!();
    }
    pub fn instantiate(
        _deps: DepsMut,
        _env: Env,
        _info: MessageInfo,
        _msg: InstantiateMsg,
    ) -> StdResult<Response> {
        Ok(Response::default())
    }
    pub fn query(_deps: Deps, _env: Env, msg: FooQueryMsg) -> StdResult<Binary> {
        match msg {
            FooQueryMsg::Foo => to_binary("foo"),
        }
    }
    let contract = ContractWrapper::new(execute, instantiate, query);
    Box::new(contract)
}

struct TestingEnv {
    router: App,
    owner: Addr,
    proxy_addr: Addr,
    foo_addr: Addr,
}

fn setup() -> TestingEnv {
    let mut router = mock_app();

    let owner = Addr::unchecked("owner");

    let proxy_id = router.store_code(contract_proxy());
    let foo_id = router.store_code(contract_foo());

    let foo_addr = router
        .instantiate_contract(
            foo_id,
            owner.clone(),
            &InstantiateMsg {
                contract_address: "".to_string(),
            },
            &[],
            "foo",
            None,
        )
        .unwrap();

    let proxy_addr = router
        .instantiate_contract(
            proxy_id,
            owner.clone(),
            &InstantiateMsg {
                contract_address: foo_addr.to_string(),
            },
            &[],
            "proxy",
            None,
        )
        .unwrap();

    TestingEnv {
        router,
        owner,
        proxy_addr,
        foo_addr,
    }
}

#[test]
fn proper_initialization() {
    setup();
}

#[test]
fn proxy() {
    let env = setup();

    let foo: String = env
        .router
        .wrap()
        .query_wasm_smart(&env.proxy_addr, &FooQueryMsg::Foo)
        .unwrap();

    assert_eq!(foo, "foo");
}
