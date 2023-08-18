mod integration_tests;

use cosmwasm_schema::QueryResponses;
use cosmwasm_std::{
    entry_point, to_binary, Addr, Deps, DepsMut, Empty, Env, Event, MessageInfo, QueryResponse,
    Response, StdError,
};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cw_storage_plus::{Item, Map};

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

    #[error("Cannot migrate from different contract type: {previous_contract}")]
    CannotMigrate { previous_contract: String },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
pub struct Phase {
    pub id: u16,
    pub contract_address: Addr,
}

pub const OWNER: Auth = Auth::new("owner");
pub const CURRENT_PHASE: Item<Phase> = Item::new("current_phase");
pub const PROPOSED_CONTRACT: Item<Addr> = Item::new("proposed_contract");
pub const PHASES: Map<u16, Addr> = Map::new("phases");

pub mod state {
    use super::*;
    /// Identical to [ocr2::state::Round], but modified to use a larger round_id to account for phase_id.
    #[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
    pub struct Round {
        pub round_id: u64,
        #[serde(with = "ocr2::state::bignum")]
        #[schemars(with = "String")]
        pub answer: i128,
        pub observations_timestamp: u32,
        pub transmission_timestamp: u32,
    }
}

pub mod msg {
    use super::*;

    #[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
    #[serde(rename_all = "snake_case")]
    pub struct InstantiateMsg {
        pub contract_address: String,
    }

    #[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
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

    #[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema, QueryResponses)]
    #[serde(rename_all = "snake_case")]
    pub enum QueryMsg {
        #[returns(u8)]
        Decimals {},
        #[returns(str)]
        Version {},
        #[returns(str)]
        Description {},
        #[returns(ocr2::state::Round)]
        RoundData { round_id: u64 },
        #[returns(ocr2::state::Round)]
        LatestRoundData {},
        #[returns(ocr2::state::Round)]
        ProposedRoundData { round_id: u32 },
        #[returns(ocr2::state::Round)]
        ProposedLatestRoundData {},
        #[returns(Addr)]
        Aggregator {},
        #[returns(u16)]
        PhaseId {},
        #[returns(Addr)]
        PhaseAggregators { phase_id: u16 },
        #[returns(Addr)]
        ProposedAggregator {},
        #[returns(Addr)]
        Owner {},
    }
}

use msg::*;
use state::*;

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:proxy-ocr2";
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

    PHASES.save(deps.storage, 1, &contract_address)?;
    CURRENT_PHASE.save(
        deps.storage,
        &Phase {
            id: 1,
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
            let current_address = CURRENT_PHASE.load(deps.storage)?.contract_address;
            Ok(Response::default().add_event(
                Event::new("propose_contract")
                    .add_attribute("current_address", current_address)
                    .add_attribute("proposed_address", address),
            ))
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

            let mut current_phase = CURRENT_PHASE.load(deps.storage)?;
            current_phase.id += 1;
            let old_address = current_phase.contract_address;
            current_phase.contract_address = address;
            CURRENT_PHASE.save(deps.storage, &current_phase)?;

            PHASES.save(
                deps.storage,
                current_phase.id,
                &current_phase.contract_address,
            )?;

            Ok(Response::default().add_event(
                Event::new("confirm_contract")
                    .add_attribute("old_address", old_address)
                    .add_attribute("new_address", current_phase.contract_address),
            ))
        }
        ExecuteMsg::TransferOwnership { to } => {
            Ok(OWNER.execute_transfer_ownership(deps, info, api.addr_validate(&to)?)?)
        }
        ExecuteMsg::AcceptOwnership => Ok(OWNER.execute_accept_ownership(deps, info)?),
    }
}

#[entry_point]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> Result<QueryResponse, ContractError> {
    match msg {
        QueryMsg::Decimals {} => {
            let contract_address = CURRENT_PHASE.load(deps.storage)?.contract_address;
            let decimals: u8 = deps
                .querier
                .query_wasm_smart(contract_address, &ocr2::msg::QueryMsg::Decimals {})?;
            Ok(to_binary(&decimals)?)
        }
        QueryMsg::Version {} => {
            let contract_address = CURRENT_PHASE.load(deps.storage)?.contract_address;
            let version: String = deps
                .querier
                .query_wasm_smart(contract_address, &ocr2::msg::QueryMsg::Version {})?;
            Ok(to_binary(&version)?)
        }
        QueryMsg::Description {} => {
            let contract_address = CURRENT_PHASE.load(deps.storage)?.contract_address;
            let description: String = deps
                .querier
                .query_wasm_smart(contract_address, &ocr2::msg::QueryMsg::Description {})?;
            Ok(to_binary(&description)?)
        }
        QueryMsg::RoundData { round_id } => {
            let (phase_id, round_id) = parse_round_id(round_id);
            let contract_address = PHASES.load(deps.storage, phase_id)?;

            let round: ocr2::state::Round = deps.querier.query_wasm_smart(
                contract_address,
                &ocr2::msg::QueryMsg::RoundData { round_id },
            )?;
            Ok(to_binary(&with_phase_id(round, phase_id))?)
        }
        QueryMsg::LatestRoundData {} => {
            let phase = CURRENT_PHASE.load(deps.storage)?;
            let round: ocr2::state::Round = deps.querier.query_wasm_smart(
                phase.contract_address,
                &ocr2::msg::QueryMsg::LatestRoundData {},
            )?;
            Ok(to_binary(&with_phase_id(round, phase.id))?)
        }
        QueryMsg::ProposedRoundData { round_id } => {
            let contract_address = PROPOSED_CONTRACT.load(deps.storage)?;
            let round: ocr2::state::Round = deps.querier.query_wasm_smart(
                contract_address,
                &ocr2::msg::QueryMsg::RoundData { round_id },
            )?;
            Ok(to_binary(&round)?)
        }
        QueryMsg::ProposedLatestRoundData {} => {
            let contract_address = PROPOSED_CONTRACT.load(deps.storage)?;
            let round: ocr2::state::Round = deps
                .querier
                .query_wasm_smart(contract_address, &ocr2::msg::QueryMsg::LatestRoundData {})?;
            Ok(to_binary(&round)?)
        }
        QueryMsg::Aggregator {} => {
            let contract_address: Addr = CURRENT_PHASE.load(deps.storage)?.contract_address;
            Ok(to_binary(&contract_address)?)
        }
        QueryMsg::PhaseId {} => {
            let phase_id: u16 = CURRENT_PHASE.load(deps.storage)?.id;
            Ok(to_binary(&phase_id)?)
        }
        QueryMsg::PhaseAggregators { phase_id } => {
            let contract_address = PHASES.load(deps.storage, phase_id)?;
            Ok(to_binary(&contract_address)?)
        }
        QueryMsg::ProposedAggregator {} => {
            let contract_address = PROPOSED_CONTRACT.load(deps.storage)?;
            Ok(to_binary(&contract_address)?)
        }
        QueryMsg::Owner {} => Ok(to_binary(&OWNER.query_owner(deps)?)?),
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(deps: DepsMut, _env: Env, _msg: Empty) -> Result<Response, ContractError> {
    let stored = cw2::get_contract_version(deps.storage)?;

    // ensure we are migrating from an allowed contract
    if stored.contract != CONTRACT_NAME {
        return Err(ContractError::CannotMigrate {
            previous_contract: stored.contract,
        });
    }

    // Update the contract version
    cw2::set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

    Ok(Response::new())
}

const PHASE_OFFSET: u32 = 32;

pub fn parse_round_id(round_id: u64) -> (u16, u32) {
    let phase_id = round_id.wrapping_shr(PHASE_OFFSET) as u16;
    let round_id = round_id as u32; // truncate higher bits
    (phase_id, round_id)
}

fn with_phase_id(round: ocr2::state::Round, phase_id: u16) -> Round {
    let round_id = ((phase_id as u64) << PHASE_OFFSET) | (round.round_id as u64);
    Round {
        round_id,
        answer: round.answer,
        observations_timestamp: round.observations_timestamp,
        transmission_timestamp: round.transmission_timestamp,
    }
}

fn validate_ownership(deps: Deps, _env: &Env, info: MessageInfo) -> Result<(), ContractError> {
    if !OWNER.is_owner(deps, &info.sender)? {
        return Err(ContractError::Unauthorized);
    }
    Ok(())
}
