#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{attr, to_binary, Deps, DepsMut, Env, MessageInfo, QueryResponse, Response};

use crate::error::ContractError;
use crate::msg::*;
use crate::state::*;

use chainlink_cosmos::msg::QueryMsg as ChainlinkQueryMsg;
use chainlink_cosmos::state::Round;

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:hello-world";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

/// Used to format the raw answer value as a human readable string.
struct Decimal {
    pub value: i128,
    pub decimals: u32,
}

impl Decimal {
    pub fn new(value: i128, decimals: u32) -> Self {
        Decimal { value, decimals }
    }
}

impl std::fmt::Display for Decimal {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let mut scaled_val = self.value.to_string();
        if scaled_val.len() <= self.decimals as usize {
            scaled_val.insert_str(
                0,
                &vec!["0"; self.decimals as usize - scaled_val.len()].join(""),
            );
            scaled_val.insert_str(0, "0.");
        } else {
            scaled_val.insert(scaled_val.len() - self.decimals as usize, '.');
        }
        f.write_str(&scaled_val)
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    let feed = deps.api.addr_validate(&msg.feed)?;

    cw2::set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

    let decimals = deps
        .querier
        .query_wasm_smart(&feed, &ChainlinkQueryMsg::Decimals {})?;

    CONFIG.save(deps.storage, &Config { feed, decimals })?;

    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::Run {} => execute_run(deps, env, info),
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> Result<QueryResponse, ContractError> {
    match msg {
        QueryMsg::Decimals {} => Ok(to_binary(&query_decimals(deps)?)?),
        QueryMsg::Round {} => Ok(to_binary(&query_round(deps)?)?),
    }
}

fn execute_run(deps: DepsMut, _env: Env, _info: MessageInfo) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;

    // Query the oracle network
    let round = deps
        .querier
        .query_wasm_smart(config.feed, &ChainlinkQueryMsg::LatestRoundData {})?;

    PRICE.save(deps.storage, &round)?;

    let decimal = Decimal::new(round.answer, u32::from(config.decimals));

    Ok(Response::new().add_attributes(vec![
        attr("price", decimal.to_string()),
        attr("observations_timestamp", round.answer.to_string()),
        attr("transmissions_timestamp", round.answer.to_string()),
    ]))
}

fn query_round(deps: Deps) -> Result<Round, ContractError> {
    let round = PRICE.load(deps.storage)?;
    Ok(round)
}

fn query_decimals(deps: Deps) -> Result<u8, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    Ok(config.decimals)
}
