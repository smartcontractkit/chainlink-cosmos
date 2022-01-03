#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    attr, to_binary, Addr, Deps, DepsMut, Env, MessageInfo, QueryResponse, Response,
};

use crate::error::ContractError;
use crate::msg::*;
use crate::require;
use crate::state::*;

use access_controller::AccessControllerContract;

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:flags";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    let raising_access_controller = deps.api.addr_validate(&msg.raising_access_controller)?;
    let lowering_access_controller = deps.api.addr_validate(&msg.lowering_access_controller)?;

    cw2::set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

    CONFIG.save(
        deps.storage,
        &Config {
            raising_access_controller: AccessControllerContract(raising_access_controller),
            lowering_access_controller: AccessControllerContract(lowering_access_controller),
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
        ExecuteMsg::RaiseFlag { subject } => execute_raise_flag(deps, env, info, subject),
        ExecuteMsg::RaiseFlags { subjects } => execute_raise_flags(deps, env, info, subjects),
        ExecuteMsg::LowerFlags { subjects } => execute_lower_flags(deps, env, info, subjects),
        ExecuteMsg::SetRaisingAccessController { rac_address } => {
            execute_set_raising_access_controller(deps, env, info, rac_address)
        }
        ExecuteMsg::TransferOwnership { to } => {
            Ok(OWNER.execute_transfer_ownership(deps, info, api.addr_validate(&to)?)?)
        }
        ExecuteMsg::AcceptOwnership => Ok(OWNER.execute_accept_ownership(deps, info)?),
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> Result<QueryResponse, ContractError> {
    match msg {
        QueryMsg::Flag { subject } => Ok(to_binary(&query_flag(deps, subject)?)?),
        QueryMsg::Flags { subjects } => Ok(to_binary(&query_flags(deps, subjects)?)?),
        QueryMsg::RaisingAccessController => {
            Ok(to_binary(&query_raising_access_controller(deps)?)?)
        }
        QueryMsg::Owner => Ok(to_binary(&OWNER.query_owner(deps)?)?),
    }
}

pub fn execute_raise_flag(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    subject: String,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    check_access(deps.as_ref(), info, &config.raising_access_controller)?;
    let subject = deps.api.addr_validate(&subject)?;

    let flag = FLAGS.may_load(deps.as_ref().storage, &subject)?.is_some();

    if flag {
        Ok(Response::new().add_attributes(vec![
            attr("action", "already raised flag"),
            attr("subject", subject),
        ]))
    } else {
        FLAGS.save(deps.storage, &subject, &())?;
        Ok(Response::new().add_attributes(vec![
            attr("action", "raised flag"),
            attr("subject", subject),
        ]))
    }
}

pub fn execute_raise_flags(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    subjects: Vec<String>,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    check_access(deps.as_ref(), info, &config.raising_access_controller)?;

    let subjects = subjects
        .iter()
        .map(|subject| deps.api.addr_validate(subject))
        .collect::<Result<Vec<Addr>, _>>()?;

    let mut attributes = Vec::with_capacity(subjects.len() * 2);

    for subject in subjects {
        let flag = FLAGS.may_load(deps.as_ref().storage, &subject)?.is_some();

        if flag {
            attributes.extend_from_slice(&[
                attr("action", "already raised flag"),
                attr("subject", subject),
            ]);
        } else {
            FLAGS.save(deps.storage, &subject, &())?;
            attributes
                .extend_from_slice(&[attr("action", "flag raised"), attr("subject", subject)]);
        }
    }
    Ok(Response::new().add_attributes(attributes))
}

pub fn execute_lower_flags(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    subjects: Vec<String>,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    check_access(deps.as_ref(), info, &config.lowering_access_controller)?;

    let subjects = subjects
        .iter()
        .map(|subject| deps.api.addr_validate(subject))
        .collect::<Result<Vec<Addr>, _>>()?;

    let mut attributes = Vec::with_capacity(subjects.len() * 2);

    for subject in subjects {
        let flag = FLAGS.may_load(deps.as_ref().storage, &subject)?.is_some();

        if flag {
            FLAGS.remove(deps.storage, &subject);
            attributes
                .extend_from_slice(&[attr("action", "flag lowered"), attr("address", subject)]);
        }
    }
    Ok(Response::new().add_attributes(attributes))
}

pub fn execute_set_raising_access_controller(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    rac_address: String,
) -> Result<Response, ContractError> {
    validate_ownership(deps.as_ref(), &env, info)?;

    let new_controller = deps.api.addr_validate(&rac_address)?;
    let mut config = CONFIG.load(deps.storage)?;
    let previous_controller = std::mem::replace(
        &mut config.raising_access_controller,
        AccessControllerContract(new_controller),
    );
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new().add_attributes(vec![
        attr("action", "raising access controller updated"),
        attr("address", rac_address),
        attr("previous", previous_controller.addr()),
    ]))
}

pub fn query_flag(deps: Deps, subject: String) -> Result<bool, ContractError> {
    // NOTE: CosmWasm queries are executed read-only, so we can't do access checks against info in queries
    let subject = deps.api.addr_validate(&subject)?;
    Ok(FLAGS.may_load(deps.storage, &subject)?.is_some())
}

pub fn query_flags(deps: Deps, subjects: Vec<String>) -> Result<Vec<bool>, ContractError> {
    // NOTE: CosmWasm queries are executed read-only, so we can't do access checks against info in queries
    let subjects = subjects
        .iter()
        .map(|subject| deps.api.addr_validate(subject))
        .collect::<Result<Vec<Addr>, _>>()?;

    let flags = subjects
        .iter()
        .map(|subject| {
            FLAGS
                .may_load(deps.storage, subject)
                .map(|result| result.is_some())
        })
        .collect::<Result<Vec<bool>, _>>()?;

    Ok(flags)
}

pub fn query_raising_access_controller(deps: Deps) -> Result<Addr, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    Ok(config.raising_access_controller.addr())
}

fn validate_ownership(deps: Deps, _env: &Env, info: MessageInfo) -> Result<(), ContractError> {
    require!(OWNER.is_owner(deps, &info.sender)?, Unauthorized);
    Ok(())
}

fn check_access(
    deps: Deps,
    info: MessageInfo,
    controller: &AccessControllerContract,
) -> Result<(), ContractError> {
    let is_owner = OWNER.is_owner(deps, &info.sender)?;

    require!(
        is_owner || controller.has_access(&deps.querier, &info.sender)?,
        Unauthorized
    );

    Ok(())
}
