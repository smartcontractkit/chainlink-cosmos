#![cfg(test)]
#![cfg(not(tarpaulin_include))]
use crate::contract::{execute, instantiate, query};
use crate::msg::{ExecuteMsg, InstantiateMsg, QueryMsg};
use cosmwasm_std::{Addr, Empty};
use cw_multi_test::{App, Contract, ContractWrapper, Executor};

fn mock_app() -> App {
    App::default()
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

mod mock {
    use cosmwasm_std::{
        to_binary, Binary, Deps, DepsMut, Empty, Env, MessageInfo, Response, StdResult,
    };
    use cw_multi_test::{Contract, ContractWrapper};

    use chainlink_cosmos::msg::QueryMsg as ChainlinkQueryMsg;
    use chainlink_cosmos::state::Round;

    pub type InstantiateMsg = ();
    pub type ExecuteMsg = ();

    pub const DECIMALS: u8 = 8;
    pub const ROUND: Round = Round {
        round_id: 1,
        answer: 1,
        observations_timestamp: 1,
        transmission_timestamp: 1,
    };

    pub fn contract() -> Box<dyn Contract<Empty>> {
        pub fn instantiate(
            _deps: DepsMut,
            _env: Env,
            _info: MessageInfo,
            _msg: InstantiateMsg,
        ) -> StdResult<Response> {
            Ok(Response::default())
        }
        pub fn execute(
            _deps: DepsMut,
            _env: Env,
            _info: MessageInfo,
            _msg: ExecuteMsg,
        ) -> StdResult<Response> {
            unimplemented!()
        }
        pub fn query(_deps: Deps, _env: Env, msg: ChainlinkQueryMsg) -> StdResult<Binary> {
            match msg {
                ChainlinkQueryMsg::Decimals {} => to_binary(&DECIMALS),
                ChainlinkQueryMsg::LatestRoundData {} => to_binary(&ROUND),
                _ => unimplemented!(),
            }
        }
        let contract = ContractWrapper::new(execute, instantiate, query);
        Box::new(contract)
    }
}

fn setup() -> Env {
    let mut router = mock_app();

    let owner = Addr::unchecked("owner");

    let hello_world_id = router.store_code(contract_hello_world());
    let proxy_id = router.store_code(mock::contract());

    let proxy_addr = router
        .instantiate_contract(proxy_id, owner.clone(), &(), &[], "hello_world", None)
        .unwrap();

    let hello_world_addr = router
        .instantiate_contract(
            hello_world_id,
            owner.clone(),
            &InstantiateMsg {
                feed: proxy_addr.to_string(),
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
    assert!(env
        .router
        .execute_contract(
            env.owner.clone(),
            env.hello_world_addr.clone(),
            &ExecuteMsg::Run {},
            &[],
        )
        .is_ok());

    // query round
    let round: chainlink_cosmos::state::Round = env
        .router
        .wrap()
        .query_wasm_smart(&env.hello_world_addr, &QueryMsg::Round {})
        .unwrap();
    assert_eq!(mock::ROUND, round);

    // query decimals
    let decimals: u8 = env
        .router
        .wrap()
        .query_wasm_smart(&env.hello_world_addr, &QueryMsg::Decimals {})
        .unwrap();
    assert_eq!(mock::DECIMALS, decimals);
}
