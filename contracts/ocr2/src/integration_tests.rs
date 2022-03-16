#![cfg(test)]
#![cfg(not(tarpaulin_include))]
use crate::contract::{execute, instantiate, query, reply};
use crate::msg::{
    ExecuteMsg, InstantiateMsg, LatestConfigDetailsResponse, LatestTransmissionDetailsResponse,
    LinkAvailableForPaymentResponse, QueryMsg,
};
use crate::state::{Billing, Round, Validator};
use crate::Decimal;
use anyhow::Result as AnyResult;
use cosmwasm_std::{to_binary, Addr, Binary, Empty, Uint128};
use cw20::Cw20Coin;
use cw_multi_test::{App, AppBuilder, AppResponse, Contract, ContractWrapper, Executor};
use deviation_flagging_validator as validator;
use ed25519_zebra::{SigningKey, VerificationKey, VerificationKeyBytes};
use rand::thread_rng;
use std::convert::TryFrom;
use std::str::FromStr;

const GIGA: u128 = 10u128.pow(9);

fn mock_app() -> App {
    AppBuilder::new().build()
}

pub fn contract_ocr2() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(execute, instantiate, query).with_reply(reply);
    Box::new(contract)
}

pub fn contract_cw20() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(
        cw20_base::contract::execute,
        cw20_base::contract::instantiate,
        cw20_base::contract::query,
    );
    Box::new(contract)
}

pub fn contract_access_controller() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(
        access_controller::contract::execute,
        access_controller::contract::instantiate,
        access_controller::contract::query,
    );
    Box::new(contract)
}

pub fn contract_validator() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(
        validator::contract::execute,
        validator::contract::instantiate,
        validator::contract::query,
    );
    Box::new(contract)
}

pub fn contract_flags() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(
        flags::contract::execute,
        flags::contract::instantiate,
        flags::contract::query,
    );
    Box::new(contract)
}

#[allow(unused)]
struct Env {
    router: App,
    owner: Addr,
    link_token_id: u64,
    billing_access_controller_addr: Addr,
    requester_access_controller_addr: Addr,
    link_token_addr: Addr,
    ocr2_addr: Addr,
    keypairs: Vec<SigningKey>,
    transmitters: Vec<String>,
    config_digest: [u8; 32],
}

const ANSWER: i128 = 1234567890;

fn transmit_report(
    env: &mut Env,
    epoch: u32,
    round: u8,
    answer: i128,
    valid_sig: bool,
) -> AnyResult<AppResponse> {
    // Build a report
    let len: u8 = 4;
    let mut report = Vec::new();
    report.extend_from_slice(&[97, 91, 43, 83]); // observations_timestamp
    let mut observers = [0; 32];
    for i in 0..len {
        observers[i as usize] = i;
    }
    report.extend_from_slice(&observers); // observers
    report.extend_from_slice(&[len]); // len
    let bytes = answer.to_be_bytes();
    for _ in 0..len {
        report.extend_from_slice(&bytes); // observation
    }
    report.extend_from_slice(&[0, 0, 0, 0, 0, 0, 0, 0, 13, 224, 182, 179, 167, 100, 0, 0]); // juels per luna (1 with 18 decimal places)

    // Generate report context
    let mut report_context = vec![0; 96];
    let (cfg_digest, ctx) = report_context.split_at_mut(32);
    let (epoch_and_round, _context) = ctx.split_at_mut(32);
    cfg_digest.copy_from_slice(&env.config_digest);

    // epoch
    epoch_and_round[27..27 + 4].clone_from_slice(&epoch.to_be_bytes());
    // round
    epoch_and_round[31] = round;

    // determine hash to sign
    use blake2::{Blake2s, Digest};
    let mut hasher = Blake2s::default();
    hasher.update((report.len() as u32).to_be_bytes());
    hasher.update(&report);
    hasher.update(&report_context);
    let hash = hasher.finalize();

    // sign with all the signers
    let signatures = env
        .keypairs
        .iter()
        .take(2)
        .map(|sk| {
            let sig = sk.sign(&hash);
            let sig_bytes: [u8; 64] = sig.into();
            let pk_bytes: [u8; 32] = VerificationKey::from(sk).into();

            let mut result = Vec::new();
            result.extend_from_slice(&pk_bytes);
            if valid_sig {
                result.extend_from_slice(&sig_bytes);
            } else {
                result.extend_from_slice(&[0u8; 64]);
            }
            Binary(result)
        })
        .collect();

    let transmitter = Addr::unchecked(env.transmitters.first().cloned().unwrap());
    let msg = ExecuteMsg::Transmit {
        report_context: Binary(report_context),
        report: Binary(report),
        signatures,
    };

    env.router
        .execute_contract(transmitter.clone(), env.ocr2_addr.clone(), &msg, &[])
}

fn setup() -> Env {
    let mut router = mock_app();

    let owner = Addr::unchecked("owner");

    let ocr2_id = router.store_code(contract_ocr2());
    let link_token_id = router.store_code(contract_cw20());
    let access_controller_id = router.store_code(contract_access_controller());

    let main_balance = Cw20Coin {
        address: owner.clone().into(),
        amount: Decimal::from_str("1000").unwrap().0,
    };

    let billing_access_controller_addr = router
        .instantiate_contract(
            access_controller_id,
            owner.clone(),
            &access_controller::msg::InstantiateMsg {},
            &[],
            "billing_access_controller",
            None,
        )
        .unwrap();

    let requester_access_controller_addr = router
        .instantiate_contract(
            access_controller_id,
            owner.clone(),
            &access_controller::msg::InstantiateMsg {},
            &[],
            "requester_access_controller",
            None,
        )
        .unwrap();

    let link_token_addr = router
        .instantiate_contract(
            link_token_id,
            owner.clone(),
            &cw20_base::msg::InstantiateMsg {
                name: String::from("Chainlink"),
                symbol: String::from("LINK"),
                decimals: 18,
                initial_balances: vec![main_balance],
                mint: None,
                marketing: None,
            },
            &[],
            "LINK",
            None,
        )
        .unwrap();

    let ocr2_addr = router
        .instantiate_contract(
            ocr2_id,
            owner.clone(),
            &InstantiateMsg {
                link_token: link_token_addr.to_string(),
                min_answer: 1i128,
                max_answer: 1_000_000_000_000i128,
                billing_access_controller: billing_access_controller_addr.to_string(),
                requester_access_controller: requester_access_controller_addr.to_string(),
                decimals: 18,
                description: "ETH/USD".to_string(),
            },
            &[],
            "OCR2",
            None,
        )
        .unwrap();

    let deposit = Decimal::from_str("1000").unwrap().0;
    // Supply contract with funds
    router
        .execute_contract(
            owner.clone(),
            link_token_addr.clone(),
            &cw20_base::msg::ExecuteMsg::Send {
                contract: ocr2_addr.to_string(),
                amount: deposit,
                msg: Binary::from(b""),
            },
            &[],
        )
        .unwrap();
    // generate a few signer keypairs
    let mut keypairs = Vec::new();
    for _ in 0..16 {
        let sk = SigningKey::new(thread_rng());
        keypairs.push(sk);
    }

    let signers = keypairs
        .iter()
        .map(|sk| Binary(VerificationKeyBytes::from(sk).as_ref().to_vec()))
        .collect();

    let transmitters = keypairs
        .iter()
        .enumerate()
        .map(|(i, _)| format!("transmitter{}", i))
        .collect::<Vec<_>>();

    let msg = ExecuteMsg::BeginProposal;
    let response = router
        .execute_contract(owner.clone(), ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // Extract the proposal id from the wasm execute event
    let execute = response
        .events
        .iter()
        .find(|event| event.ty == "wasm")
        .unwrap();
    let id = &execute
        .attributes
        .iter()
        .find(|attr| attr.key == "proposal_id")
        .unwrap()
        .value;
    let id = Uint128::new(id.parse::<u128>().unwrap());

    let msg = ExecuteMsg::ProposeConfig {
        id,
        signers,
        transmitters: transmitters.clone(),
        payees: transmitters
            .iter()
            .enumerate()
            .map(|(i, _)| format!("payee{}", i))
            .collect(),
        f: 1,
        onchain_config: Binary(vec![]),
    };

    router
        .execute_contract(owner.clone(), ocr2_addr.clone(), &msg, &[])
        .unwrap();

    let msg = ExecuteMsg::ProposeOffchainConfig {
        id,
        offchain_config_version: 1,
        offchain_config: Binary(vec![4, 5, 6]),
    };
    router
        .execute_contract(owner.clone(), ocr2_addr.clone(), &msg, &[])
        .unwrap();

    let msg = ExecuteMsg::FinalizeProposal { id };
    let response = router
        .execute_contract(owner.clone(), ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // Extract the proposal digest from the wasm execute event
    let mut digest = [0u8; 32];
    let execute = response
        .events
        .iter()
        .find(|event| event.ty == "wasm")
        .unwrap();
    let proposal_digest = &execute
        .attributes
        .iter()
        .find(|attr| attr.key == "digest")
        .unwrap()
        .value;
    hex::decode_to_slice(proposal_digest, &mut digest).unwrap();

    let msg = ExecuteMsg::AcceptProposal {
        id,
        digest: Binary(digest.to_vec()),
    };

    let response = router
        .execute_contract(owner.clone(), ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // determine the config_digest using events returned from set_config
    let mut config_digest = [0u8; 32];
    let set_config = response
        .events
        .iter()
        .find(|event| event.ty == "wasm-set_config")
        .unwrap();
    let digest = &set_config
        .attributes
        .iter()
        .find(|attr| attr.key == "latest_config_digest")
        .unwrap()
        .value;
    hex::decode_to_slice(digest, &mut config_digest).unwrap();

    Env {
        router,
        owner,
        link_token_id,
        billing_access_controller_addr,
        requester_access_controller_addr,
        link_token_addr,
        ocr2_addr,
        keypairs,
        transmitters,
        config_digest,
    }
}

#[test]
// cw3 multisig account can control cw20 admin actions
fn transmit_happy_path() {
    let mut env = setup();
    let deposit = Decimal::from_str("1000").unwrap().0;
    // expected in juels
    let observation_payment = Uint128::from(5 * GIGA);
    let reimbursement = Decimal::from_str("0.001871716").unwrap().0;

    // -- set billing

    // price in uLUNA
    let recommended_gas_price = Decimal::from_str("0.01133").unwrap();

    let msg = ExecuteMsg::SetBilling {
        config: Billing {
            recommended_gas_price_micro: recommended_gas_price,
            observation_payment_gjuels: 5,
            transmission_payment_gjuels: 0,
            ..Default::default()
        },
    };
    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // -- check config details

    let config: LatestConfigDetailsResponse = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LatestConfigDetails)
        .unwrap();

    assert_eq!(config.config_count, 1);
    assert_eq!(config.block_number, 12345);
    assert_eq!(config.config_digest, env.config_digest);

    let description: String = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::Description)
        .unwrap();
    assert_eq!(description, "ETH/USD");

    let decimals: u8 = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::Decimals)
        .unwrap();
    assert_eq!(decimals, 18);

    // Should revert
    let res = transmit_report(&mut env, 1, 1, ANSWER, false);
    assert!(res.is_err());
    assert_eq!(res.err().unwrap().to_string(), "invalid signature");

    // -- call transmit
    let res = transmit_report(&mut env, 1, 1, ANSWER, true);
    assert!(!res.is_err());

    let transmitter = Addr::unchecked(env.transmitters.first().cloned().unwrap());

    let data: Round = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LatestRoundData)
        .unwrap();
    assert_eq!(data.observations_timestamp, 1633364819);
    assert_eq!(data.transmission_timestamp, 1571797419);
    assert_eq!(data.answer, ANSWER);

    let response: LatestTransmissionDetailsResponse = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LatestTransmissionDetails)
        .unwrap();
    assert_eq!(response.round, 1);
    assert_eq!(response.latest_timestamp, data.transmission_timestamp);

    let count: u32 = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.ocr2_addr,
            &QueryMsg::OracleObservationCount {
                transmitter: transmitter.to_string(),
            },
        )
        .unwrap();
    assert_eq!(count, 1);

    let owed_payment: Uint128 = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.ocr2_addr,
            &QueryMsg::OwedPayment {
                transmitter: transmitter.to_string(),
            },
        )
        .unwrap();
    assert_eq!(owed_payment, observation_payment + reimbursement);

    let available: LinkAvailableForPaymentResponse = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LinkAvailableForPayment)
        .unwrap();

    assert_eq!(
        available.amount,
        i128::try_from(
            (deposit
                - (observation_payment * Uint128::from(env.transmitters.len() as u128))
                - reimbursement)
                .u128()
        )
        .unwrap()
    );

    let payee0 = Addr::unchecked("payee0");

    // withdraw_payment for single oracle
    let msg = ExecuteMsg::WithdrawPayment {
        transmitter: transmitter.to_string(),
    };
    env.router
        .execute_contract(payee0.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();
    let cw20::BalanceResponse { balance } = env
        .router
        .wrap()
        .query_wasm_smart(
            env.link_token_addr.to_string(),
            &cw20::Cw20QueryMsg::Balance {
                address: payee0.to_string(),
            },
        )
        .unwrap();
    assert_eq!(balance, observation_payment + reimbursement);

    // no more owed payment remaining for transmitter
    let owed_payment: Uint128 = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.ocr2_addr,
            &QueryMsg::OwedPayment {
                transmitter: transmitter.to_string(),
            },
        )
        .unwrap();
    assert_eq!(owed_payment, Uint128::zero());

    let available: LinkAvailableForPaymentResponse = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LinkAvailableForPayment)
        .unwrap();

    assert_eq!(
        available.amount,
        i128::try_from(
            (deposit
                - (observation_payment * Uint128::from(env.transmitters.len() as u128))
                - reimbursement)
                .u128()
        )
        .unwrap()
    );

    // TODO: test repeated withdrawal to check for no-op
    // https://github.com/smartcontractkit/chainlink-terra/issues/19

    // -- now trigger set_config again which should clear the state and pay out remaining oracles

    // use a new set of keypairs and signers
    let mut keypairs = Vec::new();
    for _ in 0..16 {
        let sk = SigningKey::new(thread_rng());
        keypairs.push(sk);
    }
    let signers = keypairs
        .iter()
        .map(|sk| Binary(VerificationKeyBytes::from(sk).as_ref().to_vec()))
        .collect();

    const MAX_MSG_SIZE: usize = 4 * 1024; // 4kb

    let msg = ExecuteMsg::BeginProposal;
    let response = env
        .router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // Extract the proposal id from the wasm execute event
    let execute = response
        .events
        .iter()
        .find(|event| event.ty == "wasm")
        .unwrap();
    let id = &execute
        .attributes
        .iter()
        .find(|attr| attr.key == "proposal_id")
        .unwrap()
        .value;
    let id = Uint128::new(id.parse::<u128>().unwrap());

    let msg = ExecuteMsg::ProposeConfig {
        id,
        signers,
        transmitters: env.transmitters.clone(),
        payees: env
            .transmitters
            .iter()
            .enumerate()
            .map(|(i, _)| format!("payee{}", i))
            .collect(),
        f: 5,
        onchain_config: Binary(vec![]),
    };
    assert!(to_binary(&msg).unwrap().len() <= MAX_MSG_SIZE);
    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    let msg = ExecuteMsg::ProposeOffchainConfig {
        id,
        offchain_config_version: 2,
        offchain_config: Binary(vec![1; 2165]),
    };
    assert!(to_binary(&msg).unwrap().len() <= MAX_MSG_SIZE);
    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    let msg = ExecuteMsg::FinalizeProposal { id };
    let response = env
        .router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // Extract the proposal digest from the wasm execute event
    let mut digest = [0u8; 32];
    let execute = response
        .events
        .iter()
        .find(|event| event.ty == "wasm")
        .unwrap();
    let proposal_digest = &execute
        .attributes
        .iter()
        .find(|attr| attr.key == "digest")
        .unwrap()
        .value;
    hex::decode_to_slice(proposal_digest, &mut digest).unwrap();

    let msg = ExecuteMsg::AcceptProposal {
        id,
        digest: Binary(digest.to_vec()),
    };
    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // Assert all payees were paid
    for payee in env
        .transmitters
        .iter()
        .enumerate()
        .map(|(i, _)| Addr::unchecked(format!("payee{}", i)))
    {
        let cw20::BalanceResponse { balance } = env
            .router
            .wrap()
            .query_wasm_smart(
                env.link_token_addr.to_string(),
                &cw20::Cw20QueryMsg::Balance {
                    address: payee.to_string(),
                },
            )
            .unwrap();
        if payee == "payee0" {
            assert_eq!(balance, observation_payment + reimbursement);
        } else {
            assert_eq!(balance, observation_payment);
        }
    }

    let available: LinkAvailableForPaymentResponse = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LinkAvailableForPayment)
        .unwrap();

    assert_eq!(
        available.amount,
        i128::try_from(
            (deposit
                - (observation_payment * Uint128::from(env.transmitters.len() as u128))
                - reimbursement)
                .u128()
        )
        .unwrap()
    );
}

#[test]
fn set_link_token() {
    let mut env = setup();
    let deposit = Decimal::from_str("1000").unwrap().0;
    // expected in juels
    let observation_payment = Uint128::from(5 * GIGA);
    let reimbursement = Decimal::from_str("0.001871716").unwrap().0;

    // -- set billing

    // price in uLUNA
    let recommended_gas_price = Decimal::from_str("0.01133").unwrap();

    let msg = ExecuteMsg::SetBilling {
        config: Billing {
            recommended_gas_price_micro: recommended_gas_price,
            observation_payment_gjuels: 5,
            transmission_payment_gjuels: 0,
            ..Default::default()
        },
    };

    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // -- check config details

    let config: LatestConfigDetailsResponse = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LatestConfigDetails)
        .unwrap();

    assert_eq!(config.config_count, 1);
    assert_eq!(config.block_number, 12345);
    assert_eq!(config.config_digest, env.config_digest);

    let description: String = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::Description)
        .unwrap();
    assert_eq!(description, "ETH/USD");

    let decimals: u8 = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::Decimals)
        .unwrap();
    assert_eq!(decimals, 18);

    // -- call transmit
    transmit_report(&mut env, 1, 1, ANSWER, true);

    let transmitter = Addr::unchecked(env.transmitters.first().cloned().unwrap());

    let data: Round = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LatestRoundData)
        .unwrap();
    assert_eq!(data.observations_timestamp, 1633364819);
    assert_eq!(data.transmission_timestamp, 1571797419);
    assert_eq!(data.answer, ANSWER);

    let response: LatestTransmissionDetailsResponse = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LatestTransmissionDetails)
        .unwrap();
    assert_eq!(response.round, 1);
    assert_eq!(response.latest_timestamp, data.transmission_timestamp);

    let count: u32 = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.ocr2_addr,
            &QueryMsg::OracleObservationCount {
                transmitter: transmitter.to_string(),
            },
        )
        .unwrap();
    assert_eq!(count, 1);

    let owed_payment: Uint128 = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.ocr2_addr,
            &QueryMsg::OwedPayment {
                transmitter: transmitter.to_string(),
            },
        )
        .unwrap();

    // 1 round + gas reimbursement
    assert_eq!(owed_payment, observation_payment + reimbursement);

    // ^ ---- all duplicated from transmit_happy_path()

    let new_link_token = env
        .router
        .instantiate_contract(
            env.link_token_id,
            env.owner.clone(),
            &cw20_base::msg::InstantiateMsg {
                name: String::from("Chainlink"),
                symbol: String::from("LINK"),
                decimals: 18,
                initial_balances: vec![Cw20Coin {
                    address: env.owner.to_string(),
                    amount: Uint128::from(1_000_000_000_u128),
                }],
                mint: None,
                marketing: None,
            },
            &[],
            "LINK2",
            None,
        )
        .unwrap();

    let msg = ExecuteMsg::SetLinkToken {
        link_token: new_link_token.to_string(),
        recipient: env.owner.to_string(),
    };

    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // oracles should be paid out
    for payee in env
        .transmitters
        .iter()
        .enumerate()
        .map(|(i, _)| Addr::unchecked(format!("payee{}", i)))
    {
        let cw20::BalanceResponse { balance } = env
            .router
            .wrap()
            .query_wasm_smart(
                env.link_token_addr.to_string(),
                &cw20::Cw20QueryMsg::Balance {
                    address: payee.to_string(),
                },
            )
            .unwrap();

        if payee == "payee0" {
            assert_eq!(balance, observation_payment + reimbursement);
        } else {
            assert_eq!(balance, observation_payment);
        }
    }

    // remaining balance should go to recipient
    let cw20::BalanceResponse { balance } = env
        .router
        .wrap()
        .query_wasm_smart(
            env.link_token_addr.to_string(),
            &cw20::Cw20QueryMsg::Balance {
                address: env.owner.to_string(),
            },
        )
        .unwrap();
    let expected_balance = Decimal(deposit)
        - (Decimal(Uint128::new(env.transmitters.len() as u128) * observation_payment)
            + Decimal(reimbursement));
    assert_eq!(Decimal(balance).to_string(), expected_balance.to_string());

    // token address should be changed
    let token_addr: Addr = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LinkToken)
        .unwrap();
    assert_eq!(token_addr, new_link_token);
}

#[test]
fn revert_payouts_correctly() {
    let mut env = setup();

    // set billing
    let observation_payment = Uint128::from(5 * GIGA);
    let reimbursement = Decimal::from_str("0.001871716").unwrap().0;
    let recommended_gas_price = Decimal::from_str("0.01133").unwrap();
    let msg = ExecuteMsg::SetBilling {
        config: Billing {
            recommended_gas_price_micro: recommended_gas_price,
            observation_payment_gjuels: 5,
            transmission_payment_gjuels: 0,
            ..Default::default()
        },
    };
    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // withdraw all LINK
    let available: LinkAvailableForPaymentResponse = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LinkAvailableForPayment)
        .unwrap();
    let msg = ExecuteMsg::WithdrawFunds {
        recipient: env.owner.to_string(),
        amount: Uint128::from(available.amount as u128),
    };
    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();
    let available: LinkAvailableForPaymentResponse = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LinkAvailableForPayment)
        .unwrap();
    assert_eq!(0, available.amount);

    // transmit round
    transmit_report(&mut env, 1, 1, ANSWER, true);

    // check owed balance
    let transmitter = Addr::unchecked("transmitter0");
    let payee = Addr::unchecked("payee0");

    let owed: Uint128 = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.ocr2_addr,
            &QueryMsg::OwedPayment {
                transmitter: transmitter.to_string(),
            },
        )
        .unwrap();
    assert_eq!(reimbursement + observation_payment, owed);

    // attempt to withdraw should fail without LINK token balance
    // tests the underlying `pay_oracle` function
    let msg = ExecuteMsg::WithdrawPayment {
        transmitter: transmitter.to_string(),
    };
    assert!(env
        .router
        .execute_contract(payee.clone(), env.ocr2_addr.clone(), &msg, &[])
        .is_err());

    // owed balance should not have changed
    let owed_new: Uint128 = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.ocr2_addr,
            &QueryMsg::OwedPayment {
                transmitter: transmitter.to_string(),
            },
        )
        .unwrap();
    assert_eq!(owed, owed_new);

    // attempt to change LINK token to trigger paying all oracles
    // tests the underlying `pay_oracles` function
    let new_link_token = env
        .router
        .instantiate_contract(
            env.link_token_id,
            env.owner.clone(),
            &cw20_base::msg::InstantiateMsg {
                name: String::from("Chainlink"),
                symbol: String::from("LINK"),
                decimals: 18,
                initial_balances: vec![Cw20Coin {
                    address: env.owner.to_string(),
                    amount: Uint128::from(1_000_000_000_u128),
                }],
                mint: None,
                marketing: None,
            },
            &[],
            "LINK2",
            None,
        )
        .unwrap();
    let msg = ExecuteMsg::SetLinkToken {
        link_token: new_link_token.to_string(),
        recipient: env.owner.to_string(),
    };
    assert!(env
        .router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .is_err());

    // oracles owed balance should not have changed
    for transmitter in env
        .transmitters
        .iter()
        .enumerate()
        .map(|(i, _)| Addr::unchecked(format!("transmitter{}", i)))
    {
        let balance: Uint128 = env
            .router
            .wrap()
            .query_wasm_smart(
                &env.ocr2_addr,
                &QueryMsg::OwedPayment {
                    transmitter: transmitter.to_string(),
                },
            )
            .unwrap();

        if transmitter == "transmitter0" {
            assert_eq!(balance, observation_payment + reimbursement);
        } else {
            assert_eq!(balance, observation_payment);
        }
    }
}

#[test]
fn transmit_failing_validation() {
    let mut env = setup();

    let flags_id = env.router.store_code(contract_flags());
    let validator_id = env.router.store_code(contract_validator());

    // setup flags

    let flags_addr = env
        .router
        .instantiate_contract(
            flags_id,
            env.owner.clone(),
            &flags::msg::InstantiateMsg {
                lowering_access_controller: env.billing_access_controller_addr.to_string(),
                raising_access_controller: env.billing_access_controller_addr.to_string(),
            },
            &[],
            "flags",
            None,
        )
        .unwrap();

    let validator_addr = env
        .router
        .instantiate_contract(
            validator_id,
            env.owner.clone(),
            &validator::msg::InstantiateMsg {
                flags: flags_addr.to_string(),
                flagging_threshold: 1, // 1%?
            },
            &[],
            "validator",
            None,
        )
        .unwrap();

    // Add validator to the flags access controller list
    env.router
        .execute_contract(
            env.owner.clone(),
            env.billing_access_controller_addr.clone(),
            &access_controller::msg::ExecuteMsg::AddAccess {
                address: validator_addr.to_string(),
            },
            &[],
        )
        .unwrap();

    // Configure the aggregator to use the validator
    env.router
        .execute_contract(
            env.owner.clone(),
            env.ocr2_addr.clone(),
            &ExecuteMsg::SetValidatorConfig {
                config: Some(Validator {
                    address: validator_addr.clone(),
                    gas_limit: u64::MAX,
                }),
            },
            &[],
        )
        .unwrap();

    // -- call transmit
    transmit_report(&mut env, 1, 1, 1, true);

    // check validator didn't flag
    let flagged: bool = env
        .router
        .wrap()
        .query_wasm_smart(
            &flags_addr,
            &flags::msg::QueryMsg::Flag {
                subject: env.ocr2_addr.to_string(),
            },
        )
        .unwrap();
    assert!(!flagged);

    // this should be out of threshold
    assert!(!validator::contract::is_valid(1, 1, 1000).unwrap());
    transmit_report(&mut env, 1, 2, 1000, true);

    // check validator flagged
    let flagged: bool = env
        .router
        .wrap()
        .query_wasm_smart(
            &flags_addr,
            &flags::msg::QueryMsg::Flag {
                subject: env.ocr2_addr.to_string(),
            },
        )
        .unwrap();
    assert!(flagged);

    // read latest value to be -1
    let round: Round = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LatestRoundData)
        .unwrap();
    assert_eq!(round.round_id, 2);
    assert_eq!(round.answer, 1000);
}

#[test]
fn set_billing_payout() {
    let mut env = setup();
    // expected in juels
    let observation_payment = Uint128::from(5 * GIGA);
    let reimbursement = Decimal::from_str("0.001871716").unwrap().0;

    // -- set billing
    // price in uLUNA
    let recommended_gas_price = Decimal::from_str("0.01133").unwrap();
    let msg = ExecuteMsg::SetBilling {
        config: Billing {
            recommended_gas_price_micro: recommended_gas_price,
            observation_payment_gjuels: 5,
            transmission_payment_gjuels: 0,
            ..Default::default()
        },
    };
    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // -- call transmit
    transmit_report(&mut env, 1, 1, ANSWER, true);

    // -- set billing again
    let msg = ExecuteMsg::SetBilling {
        config: Billing {
            recommended_gas_price_micro: recommended_gas_price,
            observation_payment_gjuels: 1,
            transmission_payment_gjuels: 1,
            ..Default::default()
        },
    };
    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // oracles should be paid out (same as changing LINK token)
    for payee in env
        .transmitters
        .iter()
        .enumerate()
        .map(|(i, _)| Addr::unchecked(format!("payee{}", i)))
    {
        let cw20::BalanceResponse { balance } = env
            .router
            .wrap()
            .query_wasm_smart(
                env.link_token_addr.to_string(),
                &cw20::Cw20QueryMsg::Balance {
                    address: payee.to_string(),
                },
            )
            .unwrap();

        if payee == "payee0" {
            assert_eq!(balance, observation_payment + reimbursement);
        } else {
            assert_eq!(balance, observation_payment);
        }
    }
}
