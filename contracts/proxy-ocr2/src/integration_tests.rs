#![cfg(test)]
#![cfg(not(tarpaulin_include))]
use cosmwasm_std::{Addr, Empty};
use cw_multi_test::{App, AppBuilder, Contract, ContractWrapper, Executor};

use crate::msg::{ExecuteMsg, InstantiateMsg, QueryMsg};
use crate::state::Round;
use crate::{execute, instantiate, parse_round_id, query};

fn mock_app() -> App {
    AppBuilder::new().build()
}

pub fn contract_proxy() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(execute, instantiate, query);
    Box::new(contract)
}

mod mock {
    use cosmwasm_std::{
        to_binary, Binary, Deps, DepsMut, Empty, Env, MessageInfo, Response, StdResult,
    };
    use cw_multi_test::{Contract, ContractWrapper};
    use cw_storage_plus::{Item, Map, U32Key};

    use schemars::JsonSchema;
    use serde::{Deserialize, Serialize};

    #[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
    pub struct InstantiateMsg {}

    #[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
    pub enum ExecuteMsg {
        Insert(ocr2::state::Round),
    }

    pub const LATEST_ROUND: Item<u32> = Item::new("latest_round");
    pub const ROUNDS: Map<U32Key, ocr2::state::Round> = Map::new("rounds");

    pub fn contract() -> Box<dyn Contract<Empty>> {
        pub fn execute(
            deps: DepsMut,
            _env: Env,
            _info: MessageInfo,
            msg: ExecuteMsg,
        ) -> StdResult<Response> {
            match msg {
                ExecuteMsg::Insert(round) => {
                    let round_id = LATEST_ROUND
                        .update(deps.storage, |round_id: u32| StdResult::Ok(round_id + 1))?;
                    ROUNDS.save(deps.storage, round_id.into(), &round)?;
                    Ok(Response::default())
                }
            }
        }
        pub fn instantiate(
            deps: DepsMut,
            _env: Env,
            _info: MessageInfo,
            _msg: InstantiateMsg,
        ) -> StdResult<Response> {
            LATEST_ROUND.save(deps.storage, &0)?;
            Ok(Response::default())
        }
        pub fn query(deps: Deps, _env: Env, msg: ocr2::msg::QueryMsg) -> StdResult<Binary> {
            use ocr2::msg::QueryMsg;
            match msg {
                QueryMsg::RoundData { round_id } => {
                    let round = ROUNDS.load(deps.storage, round_id.into())?;
                    to_binary(&round)
                }
                QueryMsg::LatestRoundData => {
                    let latest_round = LATEST_ROUND.load(deps.storage)?;
                    let round = ROUNDS.load(deps.storage, latest_round.into())?;
                    to_binary(&round)
                }
                _ => unimplemented!(),
            }
        }
        let contract = ContractWrapper::new(execute, instantiate, query);
        Box::new(contract)
    }
}

struct TestingEnv {
    router: App,
    owner: Addr,
    proxy_addr: Addr,
    ocr2_addr: Addr,
    ocr2_id: u64,
}

fn setup() -> TestingEnv {
    let mut router = mock_app();

    let owner = Addr::unchecked("owner");

    let proxy_id = router.store_code(contract_proxy());
    let ocr2_id = router.store_code(mock::contract());

    let ocr2_addr = router
        .instantiate_contract(
            ocr2_id,
            owner.clone(),
            &mock::InstantiateMsg {},
            &[],
            "ocr2",
            None,
        )
        .unwrap();

    let proxy_addr = router
        .instantiate_contract(
            proxy_id,
            owner.clone(),
            &InstantiateMsg {
                contract_address: ocr2_addr.to_string(),
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
        ocr2_addr,
        ocr2_id,
    }
}

#[test]
fn proper_initialization() {
    setup();
}

#[test]
fn it_works() {
    let mut env = setup();

    // insert two rounds into the current aggregator
    env.router
        .execute_contract(
            env.owner.clone(),
            env.ocr2_addr.clone(),
            &mock::ExecuteMsg::Insert(ocr2::state::Round {
                round_id: 1,
                answer: 1,
                observations_timestamp: 1,
                transmission_timestamp: 1,
            }),
            &[],
        )
        .unwrap();

    env.router
        .execute_contract(
            env.owner.clone(),
            env.ocr2_addr.clone(),
            &mock::ExecuteMsg::Insert(ocr2::state::Round {
                round_id: 2,
                answer: 2,
                observations_timestamp: 2,
                transmission_timestamp: 2,
            }),
            &[],
        )
        .unwrap();

    // query latest round
    let latest_round: Round = env
        .router
        .wrap()
        .query_wasm_smart(&env.proxy_addr, &QueryMsg::LatestRoundData)
        .unwrap();

    assert_eq!(parse_round_id(latest_round.round_id), (1, 2));

    // query by round id, it should match latest round
    let round: Round = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.proxy_addr,
            &QueryMsg::RoundData {
                round_id: latest_round.round_id,
            },
        )
        .unwrap();

    assert_eq!(round, latest_round);

    // store for later assert
    let historic_round = round;

    // instantiate a second ocr2 aggregator
    let ocr2_addr2 = env
        .router
        .instantiate_contract(
            env.ocr2_id,
            env.owner.clone(),
            &mock::InstantiateMsg {},
            &[],
            "ocr2",
            None,
        )
        .unwrap();

    // insert a rounds into the new aggregator
    env.router
        .execute_contract(
            env.owner.clone(),
            ocr2_addr2.clone(),
            &mock::ExecuteMsg::Insert(ocr2::state::Round {
                round_id: 3,
                answer: 1,
                observations_timestamp: 1,
                transmission_timestamp: 1,
            }),
            &[],
        )
        .unwrap();

    // propose it to the proxy
    env.router
        .execute_contract(
            env.owner.clone(),
            env.proxy_addr.clone(),
            &ExecuteMsg::ProposeContract {
                address: ocr2_addr2.to_string(),
            },
            &[],
        )
        .unwrap();

    // query latest round, it should still point to the old aggregator
    let latest_round: Round = env
        .router
        .wrap()
        .query_wasm_smart(&env.proxy_addr, &QueryMsg::LatestRoundData)
        .unwrap();
    assert_eq!(parse_round_id(latest_round.round_id), (1, 2));

    // but the proposed round should be newer
    let proposed_latest_round: Round = env
        .router
        .wrap()
        .query_wasm_smart(&env.proxy_addr, &QueryMsg::ProposedLatestRoundData)
        .unwrap();

    assert_eq!(proposed_latest_round.round_id, 3);

    // confirm it
    env.router
        .execute_contract(
            env.owner.clone(),
            env.proxy_addr.clone(),
            &ExecuteMsg::ConfirmContract {
                address: ocr2_addr2.to_string(),
            },
            &[],
        )
        .unwrap();

    // query latest round, it should now point to the new aggregator
    let latest_round: Round = env
        .router
        .wrap()
        .query_wasm_smart(&env.proxy_addr, &QueryMsg::LatestRoundData)
        .unwrap();
    assert_eq!(parse_round_id(latest_round.round_id), (2, 3));

    // but historic data should still work
    let round: Round = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.proxy_addr,
            &QueryMsg::RoundData {
                round_id: historic_round.round_id,
            },
        )
        .unwrap();

    assert_eq!(round, historic_round);
}
