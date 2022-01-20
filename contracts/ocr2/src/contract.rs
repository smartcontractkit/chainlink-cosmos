#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    attr, to_binary, Addr, Binary, Deps, DepsMut, Env, Event, MessageInfo, Order, Response,
    StdResult, SubMsg, Uint128, WasmMsg,
};
use cw2::set_contract_version;
use cw20::{Cw20Contract, Cw20ReceiveMsg};

use crate::error::ContractError;
use crate::msg::{
    ExecuteMsg, InstantiateMsg, LatestConfigDetailsResponse, LatestConfigDigestAndEpochResponse,
    LatestTransmissionDetailsResponse, LinkAvailableForPaymentResponse, QueryMsg,
    TransmittersResponse,
};
use crate::state::{
    config_digest_from_data, Billing, Config, Round, Transmission, Transmitter, Validator, CONFIG,
    MAX_ORACLES, OWNER, PAYEES, PROPOSED_PAYEES, SIGNERS, TRANSMISSIONS, TRANSMITTERS,
};
use crate::{decimal, require, Decimal};

use access_controller::AccessControllerContract;
use deviation_flagging_validator::msg::ExecuteMsg as ValidatorMsg;

use std::{
    convert::{TryFrom, TryInto},
    mem,
};

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:ocr2";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

// Converts a raw Map key back into Addr. Works around a cw-storage-plus limitation
fn to_addr(raw_key: Vec<u8>) -> Addr {
    Addr::unchecked(unsafe { String::from_utf8_unchecked(raw_key) })
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    let link_token = deps.api.addr_validate(&msg.link_token)?;
    let requester_access_controller = deps.api.addr_validate(&msg.requester_access_controller)?;
    let billing_access_controller = deps.api.addr_validate(&msg.billing_access_controller)?;

    OWNER.initialize(deps.storage, info.sender)?;

    let config = Config {
        link_token: Cw20Contract(link_token),
        requester_access_controller: AccessControllerContract(requester_access_controller),
        billing_access_controller: AccessControllerContract(billing_access_controller),
        min_answer: msg.min_answer,
        max_answer: msg.max_answer,

        // meta
        decimals: msg.decimals,
        description: msg.description,

        // state
        epoch: 0,
        round: 0,
        f: 0,
        n: 0,
        config_count: 0,
        latest_config_block_number: 0,
        latest_config_digest: [0u8; 32],
        latest_aggregator_round_id: 0,

        billing: Billing {
            recommended_gas_price: 0,
            observation_payment: 0,
            base_gas: None,
            gas_per_signature: None,
            gas_adjustment: None,
        },
        validator: None,
    };
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::default().add_event(
        Event::new("set_link_token").add_attribute("new_link_token", config.link_token.0),
    ))
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
        ExecuteMsg::SetConfig {
            signers,
            transmitters,
            f,
            onchain_config,
            offchain_config_version,
            offchain_config,
        } => {
            let api = &deps.api;
            let signers = signers.into_iter().map(|s| s.0).collect::<Vec<Vec<u8>>>();
            let transmitters = transmitters
                .iter()
                .map(|t| api.addr_validate(t))
                .collect::<StdResult<Vec<Addr>>>()?;
            execute_set_config(
                deps,
                env,
                info,
                signers,
                transmitters,
                f,
                onchain_config.0,
                offchain_config_version,
                offchain_config.0,
            )
        }
        ExecuteMsg::TransferOwnership { to } => {
            Ok(OWNER.execute_transfer_ownership(deps, info, api.addr_validate(&to)?)?)
        }
        ExecuteMsg::AcceptOwnership => Ok(OWNER.execute_accept_ownership(deps, info)?),
        ExecuteMsg::RequestNewRound => execute_request_new_round(deps, env, info),
        ExecuteMsg::Transmit {
            report_context,
            report,
            signatures,
        } => {
            // since we currently use Vec instead of [u8; N], verify each raw signature length first
            let signatures = signatures
                .into_iter()
                .map(|signature| signature.0.try_into())
                .collect::<Result<_, _>>()
                .map_err(|_| ContractError::InvalidInput)?;

            execute_transmit(deps, env, info, report_context.0, report.0, signatures)
        }
        ExecuteMsg::SetLinkToken {
            link_token,
            recipient,
        } => execute_set_link_token(deps, env, info, link_token, recipient),
        ExecuteMsg::Receive(msg) => execute_receive(deps, env, info, msg),
        ExecuteMsg::SetBilling { config } => execute_set_billing(deps, env, info, config),
        ExecuteMsg::SetBillingAccessController { access_controller } => {
            execute_set_billing_access_controller(deps, env, info, access_controller)
        }
        ExecuteMsg::SetRequesterAccessController { access_controller } => {
            execute_set_requester_access_controller(deps, env, info, access_controller)
        }
        ExecuteMsg::WithdrawPayment { transmitter } => {
            execute_withdraw_payment(deps, env, info, transmitter)
        }
        ExecuteMsg::WithdrawFunds { recipient, amount } => {
            execute_withdraw_funds(deps, env, info, recipient, amount)
        }
        ExecuteMsg::SetPayees { payees } => execute_set_payees(deps, env, info, payees),
        ExecuteMsg::TransferPayeeship {
            transmitter,
            proposed,
        } => execute_transfer_payeeship(deps, env, info, transmitter, proposed),
        ExecuteMsg::AcceptPayeeship { transmitter } => {
            execute_accept_payeeship(deps, env, info, transmitter)
        }
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::LatestConfigDetails => to_binary(&query_latest_config_details(deps)?),
        QueryMsg::Transmitters => to_binary(&query_transmitters(deps)?),
        QueryMsg::LatestConfigDigestAndEpoch => {
            to_binary(&query_latest_config_digest_and_epoch(deps)?)
        }
        QueryMsg::LatestTransmissionDetails => to_binary(&query_latest_transmission_details(deps)?),
        // v3
        QueryMsg::Description => to_binary(&query_description(deps)?),
        QueryMsg::Decimals => to_binary(&query_decimals(deps)?),
        QueryMsg::RoundData { round_id } => to_binary(&query_round_data(deps, round_id)?),
        QueryMsg::LatestRoundData => to_binary(&query_latest_round_data(deps)?),

        QueryMsg::LinkToken => to_binary(&query_link_token(deps)?),
        QueryMsg::Billing => to_binary(&query_billing(deps)?),
        QueryMsg::BillingAccessController => to_binary(&query_billing_access_controller(deps)?),
        QueryMsg::RequesterAccessController => to_binary(&query_requester_access_controller(deps)?),
        QueryMsg::OwedPayment { transmitter } => to_binary(&query_owed_payment(deps, transmitter)?),
        QueryMsg::LinkAvailableForPayment => {
            to_binary(&query_link_available_for_payment(deps, env)?)
        }
        QueryMsg::OracleObservationCount { transmitter } => {
            to_binary(&query_oracle_observation_count(deps, transmitter)?)
        }
        QueryMsg::Version => Ok(to_binary(CONTRACT_VERSION)?),
        QueryMsg::Owner => Ok(to_binary(&OWNER.query_owner(deps)?)?),
    }
}

pub fn execute_receive(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: Cw20ReceiveMsg,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;

    // info.sender is the address of the cw20 contract (that re-sent this message).
    require!(info.sender == config.link_token.0, Unauthorized);

    Ok(Response::default().add_event(
        Event::new("receive_funds")
            .add_attribute("sender", msg.sender)
            .add_attribute("amount", msg.amount),
    ))
}

// ---
// --- OCR2Abstract Configuration
// ---

#[allow(clippy::too_many_arguments)]
pub fn execute_set_config(
    mut deps: DepsMut,
    env: Env,
    info: MessageInfo,
    signers: Vec<Vec<u8>>,
    transmitters: Vec<Addr>,
    f: u8,
    onchain_config: Vec<u8>,
    offchain_config_version: u64,
    offchain_config: Vec<u8>,
) -> Result<Response, ContractError> {
    require!(OWNER.is_owner(deps.as_ref(), &info.sender)?, Unauthorized);

    let mut config = CONFIG.load(deps.storage)?;

    let response = Response::new().add_attribute("method", "set_config");

    let signers_len = signers.len();

    // validate new config
    require!(f != 0, InvalidInput);
    require!(signers_len <= MAX_ORACLES, TooManySigners);
    require!(transmitters.len() == signers.len(), InvalidInput);
    require!(3 * (usize::from(f)) < signers_len, InvalidInput);
    require!(onchain_config.is_empty(), InvalidInput);
    require!(!offchain_config.is_empty(), InvalidInput);

    let (_total, mut response) = pay_oracles(&mut deps, &config, response)?;

    // TODO: pay_oracles already loads all the transmitters, avoid calling TRANSMITTERS.keys
    // https://github.com/smartcontractkit/chainlink-terra/issues/27
    // Clear out oracles
    let keys: Vec<_> = TRANSMITTERS
        .keys(deps.storage, None, None, Order::Ascending)
        .collect();
    for key in keys {
        TRANSMITTERS.remove(deps.storage, &to_addr(key));
    }
    let keys: Vec<_> = SIGNERS
        .keys(deps.storage, None, None, Order::Ascending)
        .collect();
    for key in keys {
        SIGNERS.remove(deps.storage, &key);
    }

    // Update oracle set
    for transmitter in &transmitters {
        TRANSMITTERS.update(deps.storage, transmitter, |value| {
            require!(value.is_none(), RepeatedAddress);
            Ok(Transmitter {
                payment: Uint128::zero(),
                from_round_id: config.latest_aggregator_round_id,
            })
        })?;
    }
    for signer in &signers {
        SIGNERS.update(deps.storage, signer, |value| {
            require!(value.is_none(), RepeatedAddress);
            Ok(())
        })?;
    }

    // Update config
    let (previous_config_block_number, config) = {
        config.f = f;
        let previous_config_block_number = config.latest_config_block_number;
        config.latest_config_block_number = env.block.height;
        config.config_count += 1;
        config.latest_config_digest = config_digest_from_data(
            &env.block.chain_id,
            &env.contract.address,
            config.config_count,
            &signers,
            &transmitters,
            f,
            &onchain_config,
            offchain_config_version,
            &offchain_config,
        );
        config.n = signers_len as u8;

        config.epoch = 0;
        config.round = 0;
        CONFIG.save(deps.storage, &config)?;
        (previous_config_block_number, config)
    };

    let signers = signers
        .iter()
        .map(|signer| attr("signers", hex::encode(signer)));

    let transmitters = transmitters
        .iter()
        .map(|transmitter| attr("transmitters", transmitter));

    // calculate onchain_config from stored config
    let mut onchain_calc: Vec<u8> = Vec::new();
    onchain_calc.push(1);
    onchain_calc.extend_from_slice(&config.min_answer.to_be_bytes());
    onchain_calc.extend_from_slice(&config.max_answer.to_be_bytes());

    response = response.add_event(
        Event::new("set_config")
            .add_attribute(
                "previous_config_block_number",
                previous_config_block_number.to_string(),
            )
            .add_attribute(
                "latest_config_digest",
                hex::encode(config.latest_config_digest),
            )
            .add_attribute("config_count", config.config_count.to_string())
            .add_attributes(signers)
            .add_attributes(transmitters)
            .add_attribute("f", f.to_string())
            .add_attribute("onchain_config", hex::encode(onchain_calc))
            .add_attribute(
                "offchain_config_version",
                offchain_config_version.to_string(),
            )
            .add_attribute("offchain_config", hex::encode(offchain_config)),
    );

    Ok(response)
}

pub fn query_latest_config_details(deps: Deps) -> StdResult<LatestConfigDetailsResponse> {
    let config = CONFIG.load(deps.storage)?;
    Ok(LatestConfigDetailsResponse {
        config_count: config.config_count,
        block_number: config.latest_config_block_number,
        config_digest: config.latest_config_digest,
    })
}

pub fn query_transmitters(deps: Deps) -> StdResult<TransmittersResponse> {
    let addresses: Vec<_> = TRANSMITTERS
        .keys(deps.storage, None, None, Order::Ascending)
        .map(to_addr)
        .collect();
    Ok(TransmittersResponse { addresses })
}

// ---
// --- Onchain Validation
// ---

pub fn query_validator_config(deps: Deps) -> StdResult<Option<Validator>> {
    let config = CONFIG.load(deps.storage)?;
    Ok(config.validator)
}

pub fn execute_set_validator_config(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    validator: Option<Validator>,
) -> Result<Response, ContractError> {
    require!(OWNER.is_owner(deps.as_ref(), &info.sender)?, Unauthorized);
    let mut config = CONFIG.load(deps.storage)?;

    let old_validator = std::mem::replace(&mut config.validator, validator);
    CONFIG.save(deps.storage, &config)?;

    // Generate response
    let mut response = Response::default();
    if let Some(old_validator) = old_validator {
        response = response
            .add_attribute("previous_validator", old_validator.address)
            .add_attribute("previous_gas_limit", old_validator.gas_limit.to_string());
    }

    if let Some(new_validator) = config.validator {
        response = response
            .add_attribute("new_validator", new_validator.address)
            .add_attribute("new_gas_limit", new_validator.gas_limit.to_string());
    }
    Ok(response)
}

fn validate_answer(deps: Deps, config: &Config, round_id: u32, answer: i128) -> Option<SubMsg> {
    let validator = config.validator.as_ref()?;
    let previous_round_id = round_id.checked_sub(1)?;
    let previous_answer = TRANSMISSIONS
        .load(deps.storage, previous_round_id.into())
        .ok()?
        .answer;

    Some(
        SubMsg::new(WasmMsg::Execute {
            contract_addr: validator.address.to_string(),
            msg: to_binary(&ValidatorMsg::Validate {
                previous_round_id,
                previous_answer,
                round_id,
                answer,
            })
            .ok()
            .unwrap(), // maybe use Result<Option<SubMsg> _> so we can return the error from here
            funds: vec![],
        })
        .with_gas_limit(validator.gas_limit),
    )
}

// ---
// --- RequestNewRound
// ---

pub fn query_requester_access_controller(deps: Deps) -> StdResult<Addr> {
    let config = CONFIG.load(deps.storage)?;
    Ok(config.requester_access_controller.addr())
}

pub fn execute_set_requester_access_controller(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    access_controller: String,
) -> Result<Response, ContractError> {
    let access_controller = deps.api.addr_validate(&access_controller)?;

    require!(OWNER.is_owner(deps.as_ref(), &info.sender)?, Unauthorized);

    CONFIG.update(deps.storage, |mut config| -> StdResult<_> {
        config.requester_access_controller = AccessControllerContract(access_controller);
        Ok(config)
    })?;

    Ok(Response::default())
}

pub fn execute_request_new_round(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;

    let is_owner = OWNER.is_owner(deps.as_ref(), &info.sender)?;

    require!(
        is_owner
            || config
                .requester_access_controller
                .has_access(&deps.querier, &info.sender)?,
        Unauthorized
    );

    Ok(Response::default()
        .add_event(
            Event::new("round_requested")
                .add_attribute("requester", info.sender)
                .add_attribute("config_digest", hex::encode(config.latest_config_digest))
                .add_attribute("round", config.round.to_string())
                .add_attribute("epoch", config.epoch.to_string()),
        )
        .add_attribute(
            "round_id",
            (config.latest_aggregator_round_id + 1).to_string(),
        ))
}

// ---
// --- Transmission
// ---

pub fn execute_transmit(
    mut deps: DepsMut,
    env: Env,
    info: MessageInfo,
    report_context: Vec<u8>,
    raw_report: Vec<u8>,
    raw_signatures: Vec<[u8; 32 + 64]>,
) -> Result<Response, ContractError> {
    let mut response = Response::new().add_attribute("method", "transmit");

    require!(report_context.len() == 96, InvalidInput);

    // Parse the report context
    let (config_digest, ctx) = report_context.split_at(32);
    let (epoch_and_round, _extra_hash) = ctx.split_at(32);
    let config_digest: [u8; 32] = config_digest.try_into().unwrap(); // guaranteed to be 32 bytes
    let (epoch, round) = epoch_and_round[27..].split_at(4); // skip 27 byte padding, read 4 bytes epoch, then 1 byte round
    let epoch = u32::from_be_bytes(epoch.try_into().map_err(|_| ContractError::InvalidInput)?);
    let round = round[0];

    let mut config = CONFIG.load(deps.storage)?;

    response = response.add_event(
        Event::new("transmitted")
            .add_attribute("config_digest", hex::encode(config_digest))
            .add_attribute("epoch", epoch.to_string()),
    );

    // Either newer epoch, or same epoch but higher round ID
    require!((config.epoch, config.round) < (epoch, round), StaleReport);

    let mut oracle = match TRANSMITTERS.may_load(deps.storage, &info.sender)? {
        Some(oracle) => oracle,
        None => return Err(ContractError::Unauthorized),
    };

    require!(config.latest_config_digest == config_digest, DigestMismatch);

    require!(
        raw_signatures.len() == usize::from(config.f) + 1,
        WrongNumberOfSignatures
    );

    // Verify signatures attached to report
    use blake2::{Blake2s, Digest};
    let mut hasher = Blake2s::default();
    hasher.update(
        u32::try_from(raw_report.len())
            .map_err(|_| ContractError::InvalidInput)?
            .to_be_bytes(),
    );
    hasher.update(&raw_report);
    hasher.update(&report_context);
    let hash = hasher.finalize();

    let mut sigs = Vec::with_capacity(raw_signatures.len());
    let mut pkeys = Vec::with_capacity(raw_signatures.len());

    for (pubkey, signature) in raw_signatures
        .iter()
        .map(|signature| signature.split_at(32))
    {
        // Check address is present and it's a signer
        let exists = SIGNERS.has(deps.storage, pubkey);
        require!(exists, Unauthorized);
        // Check signer is unique
        match pkeys.binary_search(&pubkey) {
            // key already exists
            Ok(_index) => return Err(ContractError::RepeatedAddress),
            // not found
            Err(index) => {
                pkeys.insert(index, pubkey);
                sigs.insert(index, signature);
            }
        };
    }

    let verified = deps.api.ed25519_batch_verify(&[&hash], &sigs, &pkeys)?;
    require!(verified, InvalidSignature);

    let (juels_per_luna, response) = report(
        response,
        &mut deps,
        env,
        &info,
        &mut config,
        config_digest,
        epoch,
        round,
        &raw_report,
    )?;

    // pay transmitter the gas reimbursement
    let amount = calculate_reimbursement(&config.billing, juels_per_luna, raw_signatures.len());
    oracle.payment += amount;
    TRANSMITTERS.save(deps.storage, &info.sender, &oracle)?;

    Ok(response.add_attribute("method", "transmit"))
}

struct Report {
    pub observations: Vec<i128>,
    pub observers: [u8; MAX_ORACLES], // observer index
    pub observations_timestamp: u32,
    pub juels_per_luna: u128,
}

// NOTE: unwraps in this method can be factored out once split_array is stable
// https://github.com/rust-lang/rust/pull/83233
fn decode_report(raw_report: &[u8]) -> Result<Report, ContractError> {
    // all big endian, signed integers are two's complement
    // (uint32, 32 bytes, u8 len, len times i128, u128)

    // assert report is long enough for at least timestamp + observers + observations len
    require!(raw_report.len() >= 4 + 32 + 1, InvalidInput);

    // observations_timestamp = uint32
    let (observations_timestamp, raw_report) = raw_report.split_at(4);
    let observations_timestamp: u32 = u32::from_be_bytes(
        observations_timestamp.try_into().unwrap(), // guaranteed to be [u8; 4]
    );

    // observers = bytes32
    let (observers, raw_report) = raw_report.split_at(32);
    let observers = observers[..MAX_ORACLES].try_into().unwrap();

    // observations = len u8 + i128[len]
    let (len, raw_report) = raw_report
        .split_first()
        .ok_or(ContractError::InvalidInput)?;

    let len = usize::from(*len);

    const OBSERVATION_SIZE: usize = mem::size_of::<i128>();

    // assert the remainder of the report is long enough for N observations + juels_per_luna
    require!(
        raw_report.len() == OBSERVATION_SIZE * len + mem::size_of::<u128>(),
        InvalidInput
    );

    let (raw_observations, raw_report) = raw_report.split_at(OBSERVATION_SIZE * len);
    let observations = raw_observations
        .chunks(OBSERVATION_SIZE)
        .map(|raw| i128::from_be_bytes(raw.try_into().unwrap())) // guaranteed to be [u8; 16]
        .collect::<Vec<_>>();

    // juels per luna = u128
    let juels_per_luna = u128::from_be_bytes(
        raw_report
            .try_into()
            .map_err(|_| ContractError::InvalidInput)?,
    );

    Ok(Report {
        observations,
        observers,
        observations_timestamp,
        juels_per_luna,
    })
}

#[allow(clippy::too_many_arguments)]
fn report(
    mut response: Response,
    deps: &mut DepsMut,
    env: Env,
    info: &MessageInfo,
    config: &mut Config,
    config_digest: [u8; 32],
    epoch: u32,
    round: u8,
    raw_report: &[u8],
) -> Result<(u128, Response), ContractError> {
    let report = decode_report(raw_report)?;

    require!(report.observations.len() <= MAX_ORACLES, InvalidInput);
    require!(
        usize::from(config.f) < report.observations.len(),
        InvalidInput
    );

    config.epoch = epoch;
    config.round = round;

    // get median, validate range, store in new aggregator round
    let median = report.observations[report.observations.len() / 2];
    require!(
        config.min_answer <= median && median <= config.max_answer,
        MedianOutOfRange
    );

    config.latest_aggregator_round_id += 1;

    let observations = report
        .observations
        .iter()
        .map(|observation| attr("observations", observation.to_string()));

    // emit new transmission
    response = response.add_event(
        Event::new("new_transmission")
            .add_attributes(vec![
                attr(
                    "aggregator_round_id",
                    config.latest_aggregator_round_id.to_string(),
                ),
                attr("answer", median.to_string()),
                attr("transmitter", &info.sender),
                attr(
                    "observations_timestamp",
                    report.observations_timestamp.to_string(),
                ),
                attr("observers", hex::encode(report.observers)),
                attr("juels_per_luna", report.juels_per_luna.to_string()),
                attr("config_digest", hex::encode(config_digest)),
                attr("epoch", config.epoch.to_string()),
                attr("round", config.round.to_string()),
            ])
            .add_attributes(observations),
    );

    TRANSMISSIONS.save(
        deps.storage,
        config.latest_aggregator_round_id.into(),
        &Transmission {
            answer: median,
            observations_timestamp: report.observations_timestamp,
            transmission_timestamp: env.block.time.seconds() as u32,
        },
    )?;

    // persist vars
    CONFIG.save(deps.storage, config)?;

    if let Some(validate_msg) = validate_answer(
        deps.as_ref(),
        config,
        config.latest_aggregator_round_id,
        median,
    ) {
        response = response.add_submessage(validate_msg);
    }

    Ok((report.juels_per_luna, response))
}

pub fn query_latest_transmission_details(
    deps: Deps,
) -> StdResult<LatestTransmissionDetailsResponse> {
    let config = CONFIG.load(deps.storage)?;
    let transmission =
        TRANSMISSIONS.load(deps.storage, config.latest_aggregator_round_id.into())?;
    Ok(LatestTransmissionDetailsResponse {
        latest_config_digest: config.latest_config_digest,
        epoch: config.epoch,
        round: config.round,
        latest_answer: transmission.answer,
        latest_timestamp: transmission.transmission_timestamp,
    })
}

pub fn query_latest_config_digest_and_epoch(
    deps: Deps,
) -> StdResult<LatestConfigDigestAndEpochResponse> {
    let config = CONFIG.load(deps.storage)?;
    Ok(LatestConfigDigestAndEpochResponse {
        scan_logs: false,
        config_digest: config.latest_config_digest,
        epoch: config.epoch,
    })
}

// ---
// --- v3 AggregatorInterface
// ---

pub fn query_description(deps: Deps) -> StdResult<String> {
    let config = CONFIG.load(deps.storage)?;
    Ok(config.description)
}

pub fn query_decimals(deps: Deps) -> StdResult<u8> {
    let config = CONFIG.load(deps.storage)?;
    Ok(config.decimals)
}

pub fn query_round_data(deps: Deps, round_id: u32) -> StdResult<Round> {
    let transmission = TRANSMISSIONS.load(deps.storage, round_id.into())?;

    Ok(Round {
        round_id,
        answer: transmission.answer,
        observations_timestamp: transmission.observations_timestamp,
        transmission_timestamp: transmission.transmission_timestamp,
    })
}

pub fn query_latest_round_data(deps: Deps) -> StdResult<Round> {
    let config = CONFIG.load(deps.storage)?;
    let transmission =
        TRANSMISSIONS.load(deps.storage, config.latest_aggregator_round_id.into())?;

    Ok(Round {
        round_id: config.latest_aggregator_round_id,
        answer: transmission.answer,
        observations_timestamp: transmission.observations_timestamp,
        transmission_timestamp: transmission.transmission_timestamp,
    })
}

// ---
// --- Configurable LINK Token
// ---

pub fn execute_set_link_token(
    mut deps: DepsMut,
    env: Env,
    info: MessageInfo,
    link_token: String,
    recipient: String,
) -> Result<Response, ContractError> {
    let link_token = Cw20Contract(deps.api.addr_validate(&link_token)?);
    deps.api.addr_validate(&recipient)?;

    require!(OWNER.is_owner(deps.as_ref(), &info.sender)?, Unauthorized);

    let mut config = CONFIG.load(deps.storage)?;

    if link_token == config.link_token {
        return Ok(Response::new());
    }

    let old_link_token = config.link_token.clone();

    // Sanity check new contract is actually a token contract
    let cw20::BalanceResponse { .. } = deps.querier.query_wasm_smart(
        link_token.addr(),
        &cw20::Cw20QueryMsg::Balance {
            address: env.contract.address.to_string(),
        },
    )?;

    // retrieve current balance
    let cw20::BalanceResponse { balance } = deps.querier.query_wasm_smart(
        old_link_token.addr(),
        &cw20::Cw20QueryMsg::Balance {
            address: env.contract.address.into(),
        },
    )?;

    let (total, response) = pay_oracles(&mut deps, &config, Response::default())?;

    // We can't re-query the amount here because the payment messages are executed after the call terminates
    let remaining_balance = balance.saturating_sub(total);

    let transfer_msg = WasmMsg::Execute {
        contract_addr: old_link_token.addr().into(),
        msg: to_binary(&cw20::Cw20ExecuteMsg::Transfer {
            recipient,
            amount: remaining_balance,
        })?,
        funds: vec![],
    };

    config.link_token = link_token;
    CONFIG.save(deps.storage, &config)?;

    Ok(response
        .add_attribute("method", "set_link_token")
        .add_message(transfer_msg)
        .add_event(
            Event::new("set_link_token")
                .add_attribute("old_link_token", old_link_token.0)
                .add_attribute("new_link_token", config.link_token.0),
        ))
}

pub fn query_link_token(deps: Deps) -> StdResult<Addr> {
    let config = CONFIG.load(deps.storage)?;
    Ok(config.link_token.addr())
}

// ---
// --- BillingAccessController Management
// ---

pub fn query_billing_access_controller(deps: Deps) -> StdResult<Addr> {
    let config = CONFIG.load(deps.storage)?;
    Ok(config.billing_access_controller.addr())
}

pub fn execute_set_billing_access_controller(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    access_controller: String,
) -> Result<Response, ContractError> {
    let access_controller = deps.api.addr_validate(&access_controller)?;

    require!(OWNER.is_owner(deps.as_ref(), &info.sender)?, Unauthorized);

    CONFIG.update(deps.storage, |mut config| -> StdResult<_> {
        config.billing_access_controller = AccessControllerContract(access_controller);
        Ok(config)
    })?;

    Ok(Response::default())
}

// ---
// --- Billing
// ---

pub fn execute_set_billing(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    billing_config: Billing,
) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;

    let is_owner = OWNER.is_owner(deps.as_ref(), &info.sender)?;
    require!(
        is_owner
            || config
                .billing_access_controller
                .has_access(&deps.querier, &info.sender)?,
        Unauthorized
    );

    config.billing = billing_config;
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::default().add_event(
        Event::new("set_billing")
            .add_attribute(
                "recommended_gas_price",
                config.billing.recommended_gas_price.to_string(),
            )
            .add_attribute(
                "observation_payment",
                config.billing.observation_payment.to_string(),
            ),
    ))
}

pub fn query_billing(deps: Deps) -> StdResult<Billing> {
    let config = CONFIG.load(deps.storage)?;
    Ok(config.billing)
}

// ---
// --- Payments and Withdrawals
// ---

pub fn execute_withdraw_payment(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    transmitter: String,
) -> Result<Response, ContractError> {
    let transmitter = deps.api.addr_validate(&transmitter)?;

    let payee = match PAYEES.may_load(deps.storage, &transmitter)? {
        // validate a payee exists and matches sender
        Some(payee) if payee == info.sender => payee,
        _ => return Err(ContractError::Unauthorized),
    };

    let response = pay_oracle(deps, transmitter, payee)?;
    Ok(response.add_attribute("method", "withdraw_payment"))
}

#[inline]
fn owed_payment(config: &Config, transmitter: &Transmitter) -> StdResult<Uint128> {
    let rounds = config.latest_aggregator_round_id - transmitter.from_round_id;

    Ok(Uint128::from(config.billing.observation_payment)
        .checked_mul(rounds.into())?
        // + transmitter gas reimbursement
        .checked_add(transmitter.payment)?)
}

pub fn query_owed_payment(deps: Deps, transmitter: String) -> StdResult<Uint128> {
    let transmitter = deps.api.addr_validate(&transmitter)?;

    let transmitter = TRANSMITTERS.load(deps.storage, &transmitter)?;
    let config = CONFIG.load(deps.storage)?;
    let amount = owed_payment(&config, &transmitter)?;

    Ok(amount)
}

fn pay_oracle(deps: DepsMut, transmitter: Addr, payee: Addr) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    let state = TRANSMITTERS.load(deps.storage, &transmitter)?;
    let amount = owed_payment(&config, &state)?;

    let mut response = Response::default();

    if amount.is_zero() {
        return Ok(response);
    }

    // transmit funds
    let transfer_msg = WasmMsg::Execute {
        contract_addr: config.link_token.addr().into(),
        msg: to_binary(&cw20::Cw20ExecuteMsg::Transfer {
            recipient: payee.to_string(),
            amount,
        })?,
        funds: vec![],
    };
    response = response.add_message(transfer_msg);

    // reset reward from round & transmitter gas reimbursement
    TRANSMITTERS.save(
        deps.storage,
        &transmitter,
        &Transmitter {
            payment: Uint128::zero(),
            from_round_id: config.latest_aggregator_round_id,
        },
    )?;

    // emit event
    response = response.add_event(
        Event::new("oracle_paid")
            .add_attribute("transmitter", &transmitter)
            .add_attribute("payee", &payee)
            .add_attribute("amount", amount.to_string())
            .add_attribute("link_token", config.link_token.0),
    );

    Ok(response)
}

fn pay_oracles(
    deps: &mut DepsMut,
    config: &Config,
    response: Response,
) -> Result<(Uint128, Response), ContractError> {
    let mut msgs = Vec::with_capacity(usize::from(config.n));

    let transmitters = TRANSMITTERS
        .range(deps.storage, None, None, Order::Ascending)
        .collect::<Result<Vec<_>, _>>()?;

    let mut total = Uint128::zero();

    for (raw_key, state) in transmitters {
        let transmitter = to_addr(raw_key);

        let amount = owed_payment(config, &state)?;

        if amount.is_zero() {
            continue;
        }

        let payee = PAYEES.load(deps.storage, &transmitter)?;

        // reset reward from round & transmitter gas reimbursement
        TRANSMITTERS.save(
            deps.storage,
            &transmitter,
            &Transmitter {
                payment: Uint128::zero(),
                from_round_id: config.latest_aggregator_round_id,
            },
        )?;

        msgs.push(WasmMsg::Execute {
            contract_addr: config.link_token.addr().into(),
            msg: to_binary(&cw20::Cw20ExecuteMsg::Transfer {
                recipient: payee.to_string(),
                amount,
            })?,
            funds: vec![],
        });

        total += amount;
    }

    Ok((total, response.add_messages(msgs)))
}

pub fn execute_withdraw_funds(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    recipient: String,
    amount: Uint128,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;

    let is_owner = OWNER.is_owner(deps.as_ref(), &info.sender)?;

    require!(
        is_owner
            || config
                .billing_access_controller
                .has_access(&deps.querier, &info.sender)?,
        Unauthorized
    );

    let link_due = total_link_due(deps.as_ref())?;

    let cw20::BalanceResponse { balance } = deps.querier.query_wasm_smart(
        config.link_token.addr(),
        &cw20::Cw20QueryMsg::Balance {
            address: env.contract.address.to_string(),
        },
    )?;

    let available = balance.saturating_sub(link_due);

    let transfer_msg = WasmMsg::Execute {
        contract_addr: config.link_token.addr().into(),
        msg: to_binary(&cw20::Cw20ExecuteMsg::Transfer {
            recipient,
            amount: amount.min(available),
        })?,
        funds: vec![],
    };

    Ok(Response::default()
        .add_attribute("method", "withdraw_funds")
        .add_message(transfer_msg))
}

fn total_link_due(deps: Deps) -> StdResult<Uint128> {
    let config = CONFIG.load(deps.storage)?;

    // loop over rewards and count rounds + reimbursements
    let (rounds, reimbursements) = TRANSMITTERS
        .range(deps.storage, None, None, Order::Ascending)
        .try_fold(
            // essentially double sum() but only iterates once
            (Uint128::zero(), Uint128::zero()),
            |(rounds, reimbursements), pair| {
                pair.map(|(_, state)| {
                    (
                        Uint128::from(config.latest_aggregator_round_id - state.from_round_id)
                            + rounds,
                        reimbursements + state.payment,
                    )
                })
            },
        )?;

    let amount = Uint128::from(config.billing.observation_payment)
        .checked_mul(rounds)?
        .checked_add(reimbursements)?;

    Ok(amount)
}

pub fn query_link_available_for_payment(
    deps: Deps,
    env: Env,
) -> StdResult<LinkAvailableForPaymentResponse> {
    let config = CONFIG.load(deps.storage)?;

    let cw20::BalanceResponse { balance } = deps.querier.query_wasm_smart(
        config.link_token.addr(),
        &cw20::Cw20QueryMsg::Balance {
            address: env.contract.address.into(),
        },
    )?;

    let link_due = total_link_due(deps)?;

    use cosmwasm_std::ConversionOverflowError;

    // NOTE: entire link supply fits into a u96 so this will never overflow
    let amount = if balance > link_due {
        let abs = balance - link_due;
        i128::try_from(abs.u128())
            .map_err(|_| ConversionOverflowError::new("u128", "i128", abs.to_string()))?
    } else {
        let abs = link_due - balance;
        -i128::try_from(abs.u128())
            .map_err(|_| ConversionOverflowError::new("u128", "i128", abs.to_string()))?
    };

    Ok(LinkAvailableForPaymentResponse { amount })
}

pub fn query_oracle_observation_count(deps: Deps, transmitter: String) -> StdResult<u32> {
    let transmitter = deps.api.addr_validate(&transmitter)?;

    let transmitter = TRANSMITTERS.load(deps.storage, &transmitter)?;

    let config = CONFIG.load(deps.storage)?;

    Ok(config.latest_aggregator_round_id - transmitter.from_round_id)
}

// ---
// --- Transmitter Payment
// ---

// Returns amount in juels
fn calculate_reimbursement(
    config: &Billing,
    juels_per_luna: u128,
    signature_count: usize,
) -> Uint128 {
    let signature_count = decimal(signature_count as u64);
    let gas_per_signature = decimal(config.gas_per_signature.unwrap_or(17_000));
    let base_gas = decimal(config.base_gas.unwrap_or(84_000));
    let gas_adjustment = Decimal::percent(u64::from(config.gas_adjustment.unwrap_or(140)));

    // total gas spent
    let gas = gas_per_signature * signature_count + base_gas;
    // gas allocated seems to be about 1.4 of gas used
    let gas = gas * gas_adjustment;
    // gas cost in LUNA
    let gas_cost = Decimal(Uint128::new(u128::from(config.recommended_gas_price))) * gas;
    // total in juels
    let total = gas_cost * Decimal(Uint128::new(juels_per_luna));
    // NOTE: no stability tax is charged on transactions in LUNA

    total.0
}

// ---
// --- Payee management
// ---

// Can't be used to change payee addresses, only to initially populate them.
pub fn execute_set_payees(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    payees: Vec<(String, String)>, // (transmitter, payee)
) -> Result<Response, ContractError> {
    require!(OWNER.is_owner(deps.as_ref(), &info.sender)?, Unauthorized);

    let mut events = Vec::with_capacity(payees.len());

    let payees = payees
        .iter()
        .map(|(transmitter, payee)| -> StdResult<(Addr, Addr)> {
            Ok((
                deps.api.addr_validate(transmitter)?,
                deps.api.addr_validate(payee)?,
            ))
        })
        .collect::<StdResult<Vec<_>>>()?;

    for (transmitter, payee) in payees {
        // Set the payee unless it's already set
        PAYEES.update(deps.storage, &transmitter, |value| {
            if value.is_some() {
                return Err(ContractError::PayeeAlreadySet);
            }
            events.push(
                Event::new("payeeship_transferred")
                    .add_attribute("transmitter", &transmitter)
                    .add_attribute("current", &payee),
            );
            Ok(payee)
        })?;
    }
    Ok(Response::default().add_events(events))
}

pub fn execute_transfer_payeeship(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    transmitter: String,
    proposed: String,
) -> Result<Response, ContractError> {
    let transmitter = deps.api.addr_validate(&transmitter)?;
    let proposed = deps.api.addr_validate(&proposed)?;

    require!(info.sender != proposed, TransferToSelf);

    let current_payee = PAYEES.may_load(deps.storage, &transmitter)?;
    // only current payee can update
    require!(current_payee == Some(info.sender), Unauthorized);

    let previous_proposed = PROPOSED_PAYEES.may_load(deps.storage, &transmitter)?;

    PROPOSED_PAYEES.save(deps.storage, &transmitter, &proposed)?;

    let mut response = Response::default();

    if previous_proposed.as_ref() != Some(&proposed) {
        response = response.add_event(
            Event::new("payeeship_transfer_requested")
                .add_attribute("transmitter", &transmitter)
                .add_attribute("current", current_payee.unwrap().as_str())
                .add_attribute("proposed", &proposed),
        )
    }

    Ok(response)
}

pub fn execute_accept_payeeship(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    transmitter: String,
) -> Result<Response, ContractError> {
    let transmitter = deps.api.addr_validate(&transmitter)?;

    let proposed = PROPOSED_PAYEES.load(deps.storage, &transmitter)?;

    require!(info.sender == proposed, Unauthorized);

    let current_payee = PAYEES.may_load(deps.storage, &transmitter)?;

    PAYEES.save(deps.storage, &transmitter, &info.sender)?;
    PROPOSED_PAYEES.remove(deps.storage, &transmitter);

    Ok(Response::default().add_event(
        Event::new("payeeship_transferred")
            .add_attribute("transmitter", &transmitter)
            .add_attribute(
                "previous",
                current_payee.as_ref().map(|p| p.as_str()).unwrap_or(""),
            )
            .add_attribute("current", &info.sender),
    ))
}

// Type and version interface

#[cfg(not(tarpaulin_include))]
#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use cosmwasm_std::testing::{
        mock_dependencies, mock_env, mock_info, MockApi, MockQuerier, MockStorage,
    };
    use cosmwasm_std::OwnedDeps;

    fn setup() -> OwnedDeps<MockStorage, MockApi, MockQuerier> {
        let mut deps = mock_dependencies(&[]);

        let msg = InstantiateMsg {
            link_token: "LINK".to_string(),
            min_answer: 1i128,
            max_answer: 1_000_000_000i128,
            billing_access_controller: "billing_controller".to_string(),
            requester_access_controller: "requester_controller".to_string(),
            decimals: 18,
            description: "ETH/USD".to_string(),
        };
        let info = mock_info("owner", &[]); // &coins(1000, "earth")

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
    fn set_config() {
        let mut deps = setup();
        let owner = "owner".to_string();

        let msg = ExecuteMsg::SetConfig {
            signers: vec![
                Binary(vec![1; 64]),
                Binary(vec![2; 64]),
                Binary(vec![3; 64]),
                Binary(vec![4; 64]),
            ],
            transmitters: vec![
                "transmitter0".to_string(),
                "transmitter1".to_string(),
                "transmitter2".to_string(),
                "transmitter3".to_string(),
            ],
            f: 1,
            onchain_config: Binary(vec![]),
            offchain_config_version: 1,
            offchain_config: Binary(vec![4, 5, 6]),
        };

        let execute_info = mock_info(owner.as_str(), &[]);
        execute(deps.as_mut(), mock_env(), execute_info, msg).unwrap();
    }

    // 117 bytes
    pub const REPORT: &[u8] = &[
        97, 91, 43, 83, // observations_timestamp
        0, 1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, // observers
        4, // len
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 150, 2, 210, // observation 1
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 150, 2, 210, // observation 2
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 150, 2, 210, // observation 3
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 150, 2, 210, // observation 4
        0, 0, 0, 0, 0, 0, 0, 0, 13, 224, 182, 179, 167, 100, 0,
        0, // juels per luna (1 with 18 decimal places)
    ];

    #[test]
    fn decode_reports() {
        decode_report(&REPORT).unwrap();
    }

    #[test]
    fn payees() {
        let mut deps = setup();
        let owner = "owner".to_string();

        let msg = ExecuteMsg::SetPayees {
            payees: vec![("transmitter0".to_string(), "payee0".to_string())],
        };
        let execute_info = mock_info(&owner, &[]);
        execute(deps.as_mut(), mock_env(), execute_info, msg).unwrap();

        // setting the same payee again fails
        let msg = ExecuteMsg::SetPayees {
            payees: vec![("transmitter0".to_string(), "payee1".to_string())],
        };
        let execute_info = mock_info(&owner, &[]);
        let res = execute(deps.as_mut(), mock_env(), execute_info, msg);
        assert_eq!(res.unwrap_err(), ContractError::PayeeAlreadySet);

        // setting a different payee works
        let msg = ExecuteMsg::SetPayees {
            payees: vec![("transmitter1".to_string(), "payee1".to_string())],
        };
        let execute_info = mock_info(&owner, &[]);
        execute(deps.as_mut(), mock_env(), execute_info, msg).unwrap();

        // can't transfer to self
        let msg = ExecuteMsg::TransferPayeeship {
            transmitter: "transmitter1".to_string(),
            proposed: "payee1".to_string(),
        };
        let execute_info = mock_info("payee1", &[]);
        let res = execute(deps.as_mut(), mock_env(), execute_info, msg);
        assert_eq!(res.unwrap_err(), ContractError::TransferToSelf);

        // only payee can transfer
        let msg = ExecuteMsg::TransferPayeeship {
            transmitter: "transmitter0".to_string(),
            proposed: "payee2".to_string(),
        };
        let execute_info = mock_info("payee1", &[]);
        let res = execute(deps.as_mut(), mock_env(), execute_info, msg);
        assert_eq!(res.unwrap_err(), ContractError::Unauthorized);

        // successful transfer
        let msg = ExecuteMsg::TransferPayeeship {
            transmitter: "transmitter1".to_string(),
            proposed: "payee2".to_string(),
        };
        let execute_info = mock_info("payee1", &[]);
        execute(deps.as_mut(), mock_env(), execute_info, msg).unwrap();

        // only proposed payee can accept
        let msg = ExecuteMsg::AcceptPayeeship {
            transmitter: "transmitter1".to_string(),
        };
        let execute_info = mock_info("payee1", &[]);
        let res = execute(deps.as_mut(), mock_env(), execute_info, msg);
        assert_eq!(res.unwrap_err(), ContractError::Unauthorized);

        // succesful accept
        let msg = ExecuteMsg::AcceptPayeeship {
            transmitter: "transmitter1".to_string(),
        };
        let execute_info = mock_info("payee2", &[]);
        execute(deps.as_mut(), mock_env(), execute_info, msg).unwrap();
    }
}
