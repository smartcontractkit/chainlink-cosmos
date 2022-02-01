use cosmwasm_std::{Addr, Binary, Response};
use cosmwasm_vm::testing::{
    execute, instantiate, mock_env, mock_info, mock_instance, MockApi, MockQuerier, MockStorage,
};
use cosmwasm_vm::Instance;

use ocr2::msg::{ExecuteMsg, InstantiateMsg};
use ocr2::state::Billing;
use ocr2::Decimal;

use ed25519_zebra::{SigningKey, VerificationKey, VerificationKeyBytes};
use rand::thread_rng;

use std::str::FromStr;

// Output of cargo wasm
// NOTE: by swapping the lines below you switch between testing against the local contract build,
//   and the one built by the 'cosmwasm/workspace-builder' container.
// static WASM: &[u8] = include_bytes!("../../../target/wasm32-unknown-unknown/release/ocr2.wasm");
static WASM: &[u8] = include_bytes!("../../../artifacts/ocr2.wasm");

const OWNER: &str = "creator";

fn setup() -> Instance<MockApi, MockStorage, MockQuerier> {
    let mut deps = mock_instance(WASM, &[]);

    let msg = InstantiateMsg {
        link_token: "LINK".to_string(),
        min_answer: 0i128,
        max_answer: 100_000_000_000i128,
        billing_access_controller: "billing_controller".to_string(),
        requester_access_controller: "requester_controller".to_string(),
        decimals: 18,
        description: "ETH/USD".to_string(),
    };

    let info = mock_info(OWNER, &[]);
    let res: Response = instantiate(&mut deps, mock_env(), info, msg).unwrap();
    assert_eq!(0, res.messages.len());
    deps
}

pub const REPORT2: &[u8] = &[
    97, 91, 43, 83, // observations_timestamp
    0, 1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, // observers
    2, // len
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 150, 2, 210, // observation 1
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 150, 2, 210, // observation 2
    0, 0, 0, 0, 0, 0, 0, 0, 13, 224, 182, 179, 167, 100, 0,
    0, // juels per luna (1 with 18 decimal places)
];

#[test]
fn init_works() {
    // Sanity check that the OCR2 .wasm contract is valid and accepted by the VM
    let mut deps = setup();

    // generate a few signer keypairs
    let mut keypairs = Vec::new();

    let f = 10;
    let n = 31;

    for _ in 0..n {
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
        f,
        onchain_config: Binary(vec![]),
        offchain_config_version: 1,
        offchain_config: Binary(vec![4, 5, 6]),
    };

    let execute_info = mock_info(OWNER, &[]);
    let response: Response = execute(&mut deps, mock_env(), execute_info, msg).unwrap();

    let set_config = response
        .events
        .iter()
        .find(|event| event.ty == "set_config")
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

    // set billing
    let msg = ExecuteMsg::SetBilling {
        config: Billing {
            recommended_gas_price_uluna: Decimal::from_str("10").unwrap(),
            observation_payment_gjuels: 5,
            transmission_payment_gjuels: 0,
            ..Default::default()
        },
    };

    let execute_info = mock_info(OWNER, &[]);
    let _response: Response = execute(&mut deps, mock_env(), execute_info, msg).unwrap();

    // transmit

    // construct report
    let mut report = Vec::new();
    // observations_timestamp
    report.extend_from_slice(&[97, 91, 43, 83]);
    // observers
    report.extend_from_slice(&[
        0, 1, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 0,
    ]);
    // len
    report.push(n);
    for _ in 0..n {
        // observation
        report.extend_from_slice(&[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 73, 150, 2, 210]);
    }
    // juels per luna (1 with 18 decimal places)
    report.extend_from_slice(&[0, 0, 0, 0, 0, 0, 0, 0, 13, 224, 182, 179, 167, 100, 0, 0]);

    let mut report_context = vec![0; 96];
    let (cfg_digest, ctx) = report_context.split_at_mut(32);
    let (epoch_and_round, _context) = ctx.split_at_mut(32);
    cfg_digest.copy_from_slice(&config_digest);

    // epoch 1
    epoch_and_round[27..27 + 4].clone_from_slice(&1u32.to_be_bytes());
    // round 1
    epoch_and_round[31] = 1;

    // determine hash to sign
    use blake2::{Blake2s, Digest};
    let mut hasher = Blake2s::default();
    hasher.update((report.len() as u32).to_be_bytes());
    hasher.update(&report);
    hasher.update(&report_context);
    let hash = hasher.finalize();

    // sign with all the signers
    let signatures: Vec<_> = keypairs
        .iter()
        .take(f as usize + 1)
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

    let n = signatures.len();

    let gas_before = deps.get_gas_left();

    let transmitter = Addr::unchecked(transmitters.first().cloned().unwrap());
    let msg = ExecuteMsg::Transmit {
        report_context: Binary(report_context),
        report: Binary(report),
        signatures,
    };

    let execute_info = mock_info(transmitter.as_str(), &[]);
    let _response: Response = execute(&mut deps, mock_env(), execute_info, msg).unwrap();

    let gas_used = gas_before - deps.get_gas_left();
    // unimplemented!("gas used: {} for {} signatures", gas_used, n);
    // 1 = 403574 / 406205 / 405237
    // 2 = 447190 / 445798 / 448000
    // 3 = 512726 / 513410 / 514216
    // 4 = 577960 / 578716 / 579842

    // ~1 = 405005
    // ~2 = 446500
    // ~3 = 513450
    // ~4 = 578839

    // delta 1 = 43495
    // delta 2 = 66950
    // delta 3 = 65389
}
