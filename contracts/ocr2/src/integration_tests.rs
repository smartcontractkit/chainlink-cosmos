#![cfg(test)]
#![cfg(not(tarpaulin_include))]
use crate::contract::tests::REPORT;
use crate::contract::{execute, instantiate, query};
use crate::msg::{
    ExecuteMsg, InstantiateMsg, LatestConfigDetailsResponse, LatestTransmissionDetailsResponse,
    LinkAvailableForPaymentResponse, QueryMsg,
};
use crate::state::{Billing, Round, Transmission};
use crate::Decimal;
use cosmwasm_std::{to_binary, Addr, Binary, Empty, Uint128};
use cw20::Cw20Coin;
use cw_multi_test::{App, AppBuilder, Contract, ContractWrapper, Executor};
use ed25519_zebra::{SigningKey, VerificationKey, VerificationKeyBytes};
use rand::thread_rng;
use std::convert::TryFrom;
use std::str::FromStr;

fn mock_app() -> App {
    AppBuilder::new().build()
}

pub fn contract_ocr2() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(execute, instantiate, query);
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

fn transmit_report(env: &mut Env, epoch: u32, round: u8) {
    let report = REPORT.to_vec();

    let mut report_context = vec![0; 96];
    let (cfg_digest, ctx) = report_context.split_at_mut(32);
    let (epoch_and_round, _context) = ctx.split_at_mut(32);
    cfg_digest.copy_from_slice(&env.config_digest);

    // epoch 1
    epoch_and_round[27..27 + 4].clone_from_slice(&epoch.to_be_bytes());
    // round 1
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
            result.extend_from_slice(&sig_bytes);
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
        .unwrap();
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
            "billing_access_controller",
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
    for _ in 0..19 {
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

    let msg = ExecuteMsg::SetConfig {
        signers,
        transmitters: transmitters.clone(),
        f: 1,
        onchain_config: Binary(vec![]),
        offchain_config_version: 1,
        offchain_config: Binary(vec![4, 5, 6]),
    };
    let response = router
        .execute_contract(owner.clone(), ocr2_addr.clone(), &msg, &[])
        .unwrap();
    let set_config = response
        .events
        .iter()
        .find(|event| event.ty == "wasm-set_config")
        .unwrap();

    // determine the config_digest using events returned from set_config
    let mut config_digest = [0u8; 32];
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
    let observation_payment = Decimal::from_str("5").unwrap().0;
    let reimbursement = Decimal::from_str("0.001871716").unwrap().0;

    // -- set billing

    // price in uLUNA
    let recommended_gas_price = Decimal::from_str("0.01133").unwrap();
    // price in LUNA
    let micro = Decimal::from_str("0.000001").unwrap();
    let recommended_gas_price = (recommended_gas_price * micro).0;

    let msg = ExecuteMsg::SetBilling {
        config: Billing {
            recommended_gas_price: u64::try_from(recommended_gas_price.u128()).unwrap(),
            observation_payment: u64::try_from(observation_payment.u128()).unwrap(),
            transmission_payment: 0,
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
    transmit_report(&mut env, 1, 1);

    let transmitter = Addr::unchecked(env.transmitters.first().cloned().unwrap());

    let data: Round = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LatestRoundData)
        .unwrap();
    assert_eq!(data.observations_timestamp, 1633364819);
    assert_eq!(data.transmission_timestamp, 1571797419);
    assert_eq!(data.answer, 1234567890i128);

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

    // set_payees so we can withdraw
    let msg = ExecuteMsg::SetPayees {
        payees: env
            .transmitters
            .iter()
            .enumerate()
            .map(|(i, transmitter)| (transmitter.clone(), format!("payee{}", i)))
            .collect(),
    };
    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

    // TODO: what happens if an oracle has no payees attached?
    // https://github.com/smartcontractkit/chainlink-terra/issues/20
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
    for _ in 0..19 {
        let sk = SigningKey::new(thread_rng());
        keypairs.push(sk);
    }
    let signers = keypairs
        .iter()
        .map(|sk| Binary(VerificationKeyBytes::from(sk).as_ref().to_vec()))
        .collect();

    let msg = ExecuteMsg::SetConfig {
        signers,
        transmitters: env.transmitters.clone(),
        f: 6,
        onchain_config: Binary(vec![]),
        offchain_config_version: 2,
        offchain_config: Binary(vec![1; 2165]),
    };

    const MAX_MSG_SIZE: usize = 4 * 1024; // 4kb
    assert!(to_binary(&msg).unwrap().len() <= MAX_MSG_SIZE);

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
    let observation_payment = Decimal::from_str("5").unwrap().0;
    let reimbursement = Decimal::from_str("0.001871716").unwrap().0;

    // -- set billing

    // price in uLUNA
    let recommended_gas_price = Decimal::from_str("0.01133").unwrap();
    // price in LUNA
    let micro = Decimal::from_str("0.000001").unwrap();
    let recommended_gas_price = (recommended_gas_price * micro).0;

    let msg = ExecuteMsg::SetBilling {
        config: Billing {
            recommended_gas_price: u64::try_from(recommended_gas_price.u128()).unwrap(),
            observation_payment: u64::try_from(observation_payment.u128()).unwrap(),
            transmission_payment: 0,
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
    transmit_report(&mut env, 1, 1);

    let transmitter = Addr::unchecked(env.transmitters.first().cloned().unwrap());

    let data: Round = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LatestRoundData)
        .unwrap();
    assert_eq!(data.observations_timestamp, 1633364819);
    assert_eq!(data.transmission_timestamp, 1571797419);
    assert_eq!(data.answer, 1234567890i128);

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

    // set_payees so we can withdraw
    let msg = ExecuteMsg::SetPayees {
        payees: env
            .transmitters
            .iter()
            .enumerate()
            .map(|(i, transmitter)| (transmitter.clone(), format!("payee{}", i)))
            .collect(),
    };
    env.router
        .execute_contract(env.owner.clone(), env.ocr2_addr.clone(), &msg, &[])
        .unwrap();

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
                    amount: Uint128::from(1_000_000_000 as u128),
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
        - (Decimal(Uint128::new(19) * observation_payment) + Decimal(reimbursement));
    assert_eq!(Decimal(balance).to_string(), expected_balance.to_string());

    // token address should be changed
    let token_addr: Addr = env
        .router
        .wrap()
        .query_wasm_smart(&env.ocr2_addr, &QueryMsg::LinkToken)
        .unwrap();
    assert_eq!(token_addr, new_link_token);
}
