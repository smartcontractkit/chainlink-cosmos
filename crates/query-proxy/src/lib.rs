mod integration_tests;

use cosmwasm_std::{Addr, StdError};
use cw_storage_plus::{Item, Map, U16Key};

use schemars::JsonSchema;
use serde::{Deserialize, Serialize};
use thiserror::Error;

use owned::Auth;

#[derive(Error, Debug)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("{0}")]
    Owned(#[from] owned::Error),

    #[error("Unauthorized")]
    Unauthorized,

    #[error("Invalid")]
    Invalid,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Phase {
    pub id: u16,
    pub contract_address: Addr,
}

pub const OWNER: Auth = Auth::new("owner");
pub const CURRENT_PHASE: Item<Phase> = Item::new("current_phase");
pub const PROPOSED_CONTRACT: Item<Addr> = Item::new("proposed_contract");
pub const PHASES: Map<U16Key, Addr> = Map::new("phases");

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {
    pub contract_address: String,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    ProposeContract {
        address: String,
    },
    ConfirmContract {
        address: String,
    },
    /// Initiate contract ownership transfer to another address.
    /// Can be used only by owner
    TransferOwnership {
        /// Address to transfer ownership to
        to: String,
    },
    /// Finish contract ownership transfer. Can be used only by pending owner
    AcceptOwnership,
}

#[macro_export]
macro_rules! contract {
    ($query_msg:path) => {
        use cosmwasm_std::{
            entry_point, to_binary, Deps, DepsMut, Env, MessageInfo, QueryResponse, Response,
            StdResult,
        };

        pub use $crate::{ContractError, Phase, CURRENT_PHASE, OWNER, PHASES, PROPOSED_CONTRACT};

        pub mod msg {
            pub type QueryMsg = $query_msg;
            pub use $crate::{ExecuteMsg, InstantiateMsg};
            // Wraps a query. Passed through transparently.
        }

        use msg::*;

        // version info for migration info
        const CONTRACT_NAME: &str = "crates.io:query-proxy";
        const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

        #[entry_point]
        pub fn instantiate(
            deps: DepsMut,
            _env: Env,
            info: MessageInfo,
            msg: InstantiateMsg,
        ) -> Result<Response, ContractError> {
            let contract_address = deps.api.addr_validate(&msg.contract_address)?;

            cw2::set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

            OWNER.initialize(deps.storage, info.sender)?;

            PHASES.save(deps.storage, 0.into(), &contract_address)?;
            CURRENT_PHASE.save(
                deps.storage,
                &Phase {
                    id: 0,
                    contract_address,
                },
            )?;

            Ok(Response::default())
        }

        #[entry_point]
        pub fn execute(
            deps: DepsMut,
            env: Env,
            info: MessageInfo,
            msg: ExecuteMsg,
        ) -> Result<Response, ContractError> {
            let api = deps.api;
            match msg {
                ExecuteMsg::ProposeContract { address } => {
                    let address = deps.api.addr_validate(&address)?;
                    validate_ownership(deps.as_ref(), &env, info)?;
                    PROPOSED_CONTRACT.save(deps.storage, &address)?;
                    Ok(Response::default())
                }
                ExecuteMsg::ConfirmContract { address } => {
                    let address = deps.api.addr_validate(&address)?;
                    validate_ownership(deps.as_ref(), &env, info)?;

                    // Validate the address was actually proposed previously
                    let proposed = PROPOSED_CONTRACT.load(deps.storage)?;
                    if proposed != address {
                        return Err(ContractError::Invalid);
                    }

                    // Update state
                    PROPOSED_CONTRACT.remove(deps.storage);
                    let current_phase =
                        CURRENT_PHASE.update(deps.storage, |mut phase| -> StdResult<Phase> {
                            phase.id += 1;
                            phase.contract_address = address;
                            Ok(phase)
                        })?;
                    PHASES.save(
                        deps.storage,
                        current_phase.id.into(),
                        &current_phase.contract_address,
                    )?;

                    Ok(Response::default())
                }
                ExecuteMsg::TransferOwnership { to } => {
                    Ok(OWNER.execute_transfer_ownership(deps, info, api.addr_validate(&to)?)?)
                }
                ExecuteMsg::AcceptOwnership => Ok(OWNER.execute_accept_ownership(deps, info)?),
            }
        }

        /// Delegate queries to the current contract. This is slightly less efficient than it could be if we used a raw entrypoint because
        /// we deserialize the msg, then serialize it again, but that required us to copy a lot of private modules from cosmwasm_std.
        #[entry_point]
        pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> Result<QueryResponse, ContractError> {
            let contract_address = CURRENT_PHASE.load(deps.storage)?.contract_address;

            // This recreates QuerierWrapper::query/custom_query without to_binary on the result
            use cosmwasm_std::{
                to_vec, ContractResult, Empty, QueryRequest, StdError, SystemResult, WasmQuery,
            };
            let request: QueryRequest<Empty> = WasmQuery::Smart {
                contract_addr: contract_address.into(),
                msg: to_binary(&msg)?,
            }
            .into();
            let raw = to_vec(&request).map_err(|serialize_err| {
                StdError::generic_err(format!("Serializing QueryRequest: {}", serialize_err))
            })?;
            match deps.querier.raw_query(&raw) {
                SystemResult::Err(system_err) => Err(StdError::generic_err(format!(
                    "Querier system error: {}",
                    system_err
                ))
                .into()),
                SystemResult::Ok(ContractResult::Err(contract_err)) => Err(StdError::generic_err(
                    format!("Querier contract error: {}", contract_err),
                )
                .into()),
                SystemResult::Ok(ContractResult::Ok(value)) => Ok(value),
            }
        }

        fn validate_ownership(
            deps: Deps,
            _env: &Env,
            info: MessageInfo,
        ) -> Result<(), ContractError> {
            if !OWNER.is_owner(deps, &info.sender)? {
                return Err(ContractError::Unauthorized);
            }
            Ok(())
        }
    };
}
