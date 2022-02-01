#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{to_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult};
use cw2::set_contract_version;

use crate::error::ContractError;
use crate::msg::{ExecuteMsg, InstantiateMsg, QueryMsg};
use crate::require;
use crate::state::{ACCESS, OWNER};

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:access-controller";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    _msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
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
        ExecuteMsg::AddAccess { address } => execute_add_access(deps, env, info, address),
        ExecuteMsg::RemoveAccess { address } => execute_remove_access(deps, env, info, address),
        ExecuteMsg::TransferOwnership { to } => {
            Ok(OWNER.execute_transfer_ownership(deps, info, api.addr_validate(&to)?)?)
        }
        ExecuteMsg::AcceptOwnership => Ok(OWNER.execute_accept_ownership(deps, info)?),
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::HasAccess { address } => to_binary(&query_has_access(deps, address)?),
        QueryMsg::Owner => Ok(to_binary(&OWNER.query_owner(deps)?)?),
    }
}

pub fn execute_add_access(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    address: String,
) -> Result<Response, ContractError> {
    require!(OWNER.is_owner(deps.as_ref(), &info.sender)?, Unauthorized);

    let address = deps.api.addr_validate(&address)?;
    ACCESS.save(deps.storage, &address, &())?;

    Ok(Response::default())
}

pub fn execute_remove_access(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    address: String,
) -> Result<Response, ContractError> {
    require!(OWNER.is_owner(deps.as_ref(), &info.sender)?, Unauthorized);

    let address = deps.api.addr_validate(&address)?;
    ACCESS.remove(deps.storage, &address);

    Ok(Response::default())
}

pub fn query_has_access(deps: Deps, address: String) -> StdResult<bool> {
    let address = deps.api.addr_validate(&address)?;
    let access = ACCESS.may_load(deps.storage, &address)?;
    Ok(access.is_some())
}

#[cfg(not(tarpaulin_include))]
#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{
        mock_dependencies, mock_env, mock_info, MockApi, MockQuerier, MockStorage,
    };
    use cosmwasm_std::{from_binary, OwnedDeps};

    fn setup() -> OwnedDeps<MockStorage, MockApi, MockQuerier> {
        let mut deps = mock_dependencies(&[]);

        let msg = InstantiateMsg {};
        let info = mock_info("owner", &[]);

        // we can just call .unwrap() to assert this was a success
        let res = instantiate(deps.as_mut(), mock_env(), info, msg).unwrap();
        assert_eq!(0, res.messages.len());
        deps
    }

    #[test]
    fn proper_initialization() {
        setup();
    }

    #[test]
    fn it_works() {
        let mut deps = setup();
        let owner = "owner".to_string();

        // add access to user0
        let msg = ExecuteMsg::AddAccess {
            address: "user0".to_string(),
        };
        let execute_info = mock_info(owner.as_str(), &[]);
        execute(deps.as_mut(), mock_env(), execute_info, msg).unwrap();

        // user0 has access
        let msg = QueryMsg::HasAccess {
            address: "user0".to_string(),
        };
        let raw = query(deps.as_ref(), mock_env(), msg).unwrap();
        let access: bool = from_binary(&raw).unwrap();
        assert_eq!(access, true);

        // user1 doesn't have access
        let msg = QueryMsg::HasAccess {
            address: "user1".to_string(),
        };
        let raw = query(deps.as_ref(), mock_env(), msg).unwrap();
        let access: bool = from_binary(&raw).unwrap();
        assert_eq!(access, false);

        // now remove access
        let msg = ExecuteMsg::RemoveAccess {
            address: "user0".to_string(),
        };
        let execute_info = mock_info(owner.as_str(), &[]);
        execute(deps.as_mut(), mock_env(), execute_info, msg).unwrap();

        // user0 no longer has access
        let msg = QueryMsg::HasAccess {
            address: "user0".to_string(),
        };
        let raw = query(deps.as_ref(), mock_env(), msg).unwrap();
        let access: bool = from_binary(&raw).unwrap();
        assert_eq!(access, false);
    }
}
