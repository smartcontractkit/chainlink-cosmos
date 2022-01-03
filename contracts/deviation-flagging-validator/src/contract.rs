#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_binary, Addr, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult, WasmMsg,
};

use crate::error::ContractError;
use crate::msg::*;
use crate::state::*;

use flags::msg::ExecuteMsg as FlagsMsg;

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:deviation-flagging-validator";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

static THRESHOLD_MULTIPLIER: u128 = 100000;

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    let flags = deps.api.addr_validate(&msg.flags)?;

    cw2::set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

    CONFIG.save(
        deps.storage,
        &State {
            flags,
            flagging_threshold: msg.flagging_threshold,
        },
    )?;

    OWNER.initialize(deps.storage, info.sender)?;

    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    let api = deps.api;
    match msg {
        ExecuteMsg::SetFlagsAddress { flags } => execute_set_flags_address(deps, env, info, flags),
        ExecuteMsg::SetFlaggingThreshold { threshold } => {
            execute_set_flagging_threshold(deps, env, info, threshold)
        }
        ExecuteMsg::Validate {
            previous_round_id,
            previous_answer,
            round_id,
            answer,
        } => execute_validate(
            deps,
            env,
            info,
            previous_round_id,
            previous_answer,
            round_id,
            answer,
        ),
        ExecuteMsg::TransferOwnership { to } => {
            Ok(OWNER.execute_transfer_ownership(deps, info, api.addr_validate(&to)?)?)
        }
        ExecuteMsg::AcceptOwnership => Ok(OWNER.execute_accept_ownership(deps, info)?),
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::IsValid {
            previous_answer,
            answer,
        } => {
            let flagging_threshold = CONFIG.load(deps.storage)?.flagging_threshold;
            to_binary(&is_valid(flagging_threshold, previous_answer, answer)?)
        }
        QueryMsg::FlaggingThreshold => to_binary(&query_flagging_threshold(deps)?),
        QueryMsg::Owner => Ok(to_binary(&OWNER.query_owner(deps)?)?),
    }
}

pub fn execute_validate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    _previous_round_id: u32,
    previous_answer: i128,
    _round_id: u32,
    answer: i128,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    let is_valid = is_valid(config.flagging_threshold, previous_answer, answer)?;
    let mut response = Response::default();

    if !is_valid {
        response = response.add_message(WasmMsg::Execute {
            contract_addr: config.flags.to_string(),
            msg: to_binary(&FlagsMsg::RaiseFlag {
                subject: info.sender.to_string(),
            })?,
            funds: vec![],
        })
    }

    Ok(response
        .add_attribute("action", "validate")
        .add_attribute("is_valid", is_valid.to_string())
        .set_data(to_binary(&is_valid)?))
}

pub fn execute_set_flags_address(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    flags: Addr,
) -> Result<Response, ContractError> {
    validate_ownership(deps.as_ref(), &env, info)?;
    let mut config = CONFIG.load(deps.storage)?;
    let mut response = Response::default();

    if config.flags != flags {
        let previous = std::mem::replace(&mut config.flags, flags);
        CONFIG.save(deps.storage, &config)?;
        response = response
            .add_attribute("action", "flags_address_updated")
            .add_attribute("previous", previous)
    }

    Ok(response)
}

pub fn execute_set_flagging_threshold(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    threshold: u32,
) -> Result<Response, ContractError> {
    validate_ownership(deps.as_ref(), &env, info)?;
    let mut config = CONFIG.load(deps.storage)?;
    let mut response = Response::default();

    if config.flagging_threshold != threshold {
        let previous = std::mem::replace(&mut config.flagging_threshold, threshold);
        CONFIG.save(deps.storage, &config)?;
        response = response
            .add_attribute("action", "flagging_threshold_updated")
            .add_attribute("previous", previous.to_string())
            .add_attribute("current", threshold.to_string());
    }

    Ok(response)
}

fn is_valid(flagging_threshold: u32, previous_answer: i128, answer: i128) -> StdResult<bool> {
    if previous_answer == 0i128 {
        return Ok(true);
    }

    // https://github.com/rust-lang/rust/issues/89492
    fn abs_diff(slf: i128, other: i128) -> u128 {
        if slf < other {
            (other as u128).wrapping_sub(slf as u128)
        } else {
            (slf as u128).wrapping_sub(other as u128)
        }
    }
    let change = abs_diff(previous_answer, answer);
    let ratio_numerator = match change.checked_mul(THRESHOLD_MULTIPLIER) {
        Some(ratio_numerator) => ratio_numerator,
        None => return Ok(false),
    };
    let ratio = ratio_numerator / previous_answer.unsigned_abs();
    Ok(ratio <= u128::from(flagging_threshold))
}

pub fn query_flagging_threshold(deps: Deps) -> StdResult<FlaggingThresholdResponse> {
    let flagging_threshold = CONFIG.load(deps.storage)?.flagging_threshold;
    Ok(FlaggingThresholdResponse {
        threshold: flagging_threshold,
    })
}

fn validate_ownership(deps: Deps, _env: &Env, info: MessageInfo) -> Result<(), ContractError> {
    if !OWNER.is_owner(deps, &info.sender)? {
        return Err(ContractError::Unauthorized);
    }
    Ok(())
}

#[cfg(test)]
#[cfg(not(tarpaulin_include))]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{mock_dependencies, mock_env, mock_info};
    use cosmwasm_std::{attr, coins, Api};

    #[test]
    fn proper_initialization() {
        let mut deps = mock_dependencies(&[]);

        let msg = InstantiateMsg {
            flags: "flags".to_string(),
            flagging_threshold: 100000,
        };
        let info = mock_info("creator", &coins(1000, "earth"));

        // we can just call .unwrap() to assert this was a success
        let res = instantiate(deps.as_mut(), mock_env(), info, msg).unwrap();
        assert_eq!(0, res.messages.len());
    }

    #[test]
    fn setting_flags_address() {
        let mut deps = mock_dependencies(&[]);

        let msg = InstantiateMsg {
            flags: "flags".to_string(),
            flagging_threshold: 100000,
        };
        let info = mock_info("creator", &coins(1000, "earth"));

        // we can just call .unwrap() to assert this was a success
        let res = instantiate(deps.as_mut(), mock_env(), info.clone(), msg).unwrap();
        assert_eq!(0, res.messages.len());

        let new_flags = deps.api.addr_validate("new_flags").unwrap();
        let msg = ExecuteMsg::SetFlagsAddress {
            flags: new_flags.clone(),
        };
        let res = execute(deps.as_mut(), mock_env(), info, msg).unwrap();
        assert_eq!(0, res.messages.len());

        let flag_addr = CONFIG.load(&deps.storage).unwrap().flags;
        assert_eq!(new_flags, flag_addr);
    }

    #[test]
    fn setting_threshold() {
        let mut deps = mock_dependencies(&[]);

        let msg = InstantiateMsg {
            flags: "flags".to_string(),
            flagging_threshold: 100000,
        };
        let info = mock_info("creator", &coins(1000, "earth"));

        // we can just call .unwrap() to assert this was a success
        let res = instantiate(deps.as_mut(), mock_env(), info.clone(), msg).unwrap();
        assert_eq!(0, res.messages.len());

        let msg = ExecuteMsg::SetFlaggingThreshold { threshold: 1000 };

        let _res = execute(deps.as_mut(), mock_env(), info, msg).unwrap();

        let threshold = CONFIG.load(&deps.storage).unwrap().flagging_threshold;
        assert_eq!(1000, threshold);
    }

    #[test]
    fn is_valid_gives_right_response() {
        let flagging_threshold = 80000;

        let previous_answer = 100;
        let answer = 5;
        let check_valid = is_valid(flagging_threshold, previous_answer, answer).unwrap();
        assert_eq!(false, check_valid);

        // this input should return true
        let previous_answer = 3;
        let answer = 1;
        let check_valid = is_valid(flagging_threshold, previous_answer, answer).unwrap();
        assert_eq!(true, check_valid);

        // should return true if previous_answer is 0
        let previous_answer = 0;
        let answer = 5;
        let check_valid = is_valid(flagging_threshold, previous_answer, answer).unwrap();
        assert_eq!(true, check_valid);
    }

    #[test]
    fn validate() {
        let mut deps = mock_dependencies(&[]);

        let msg = InstantiateMsg {
            flags: "flags".to_string(),
            flagging_threshold: 80000,
        };
        let info = mock_info("creator", &coins(1000, "earth"));

        // we can just call .unwrap() to assert this was a success
        let res = instantiate(deps.as_mut(), mock_env(), info.clone(), msg).unwrap();
        assert_eq!(0, res.messages.len());

        let msg = ExecuteMsg::Validate {
            previous_round_id: 2,
            previous_answer: 3,
            answer: 1,
            round_id: 3,
        };

        // the case if validate is true
        let res = execute(deps.as_mut(), mock_env(), info.clone(), msg).unwrap();
        assert_eq!(
            vec![
                attr("action", "validate"),
                attr("is_valid", true.to_string())
            ],
            res.attributes
        );

        let msg = ExecuteMsg::Validate {
            previous_round_id: 2,
            previous_answer: 100,
            answer: 5,
            round_id: 3,
        };
        let res = execute(deps.as_mut(), mock_env(), info.clone(), msg).unwrap();
        assert_eq!(
            vec![
                attr("action", "validate"),
                attr("is_valid", false.to_string())
            ],
            res.attributes
        );

        // should not panic from overflow
        let msg = ExecuteMsg::Validate {
            previous_round_id: 2,
            previous_answer: i128::MIN,
            round_id: 3,
            answer: i128::MAX,
        };
        let res = execute(deps.as_mut(), mock_env(), info, msg).unwrap();
        assert_eq!(
            vec![
                attr("action", "validate"),
                attr("is_valid", false.to_string())
            ],
            res.attributes
        );
    }

    #[test]
    fn test_query_flagging_threshold() {
        let mut deps = mock_dependencies(&[]);

        let msg = InstantiateMsg {
            flags: "flags".to_string(),
            flagging_threshold: 80000,
        };
        let info = mock_info("creator", &coins(1000, "earth"));

        // we can just call .unwrap() to assert this was a success
        let res = instantiate(deps.as_mut(), mock_env(), info.clone(), msg).unwrap();
        assert_eq!(0, res.messages.len());

        let flagging_threshold: u32 = query_flagging_threshold(deps.as_ref()).unwrap().threshold;
        assert_eq!(80000 as u32, flagging_threshold);
    }
}
