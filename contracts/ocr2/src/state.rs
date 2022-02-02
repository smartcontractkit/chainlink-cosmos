use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use access_controller::AccessControllerContract;
use cosmwasm_std::{Addr, Uint128};
use cw20::Cw20Contract;
use cw_storage_plus::{Item, Map, U32Key};
use owned::Auth;

use crate::Decimal;

/// Maximum number of oracles the offchain reporting protocol is designed for
pub const MAX_ORACLES: usize = 31;

pub mod bignum {
    use serde::{self, Deserialize, Deserializer, Serializer};

    pub fn serialize<S>(bigint: &i128, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(&bigint.to_string())
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<i128, D::Error>
    where
        D: Deserializer<'de>,
    {
        let str = String::deserialize(deserializer)?;
        str::parse::<i128>(&str).map_err(serde::de::Error::custom)
    }
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Validator {
    pub address: Addr,
    pub gas_limit: u64,
}

#[derive(Serialize, Deserialize, Clone, Default, Debug, PartialEq, JsonSchema)]
pub struct Billing {
    /// Should match https://fcd.terra.dev/v1/txs/gas_prices
    pub recommended_gas_price_uluna: Decimal,
    pub observation_payment_gjuels: u64,
    pub transmission_payment_gjuels: u64,
    pub gas_base: Option<u64>,
    pub gas_per_signature: Option<u64>,
    /// In percent
    pub gas_adjustment: Option<u8>,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Config {
    pub link_token: Cw20Contract,
    pub requester_access_controller: AccessControllerContract,
    pub billing_access_controller: AccessControllerContract,

    #[serde(with = "bignum")]
    #[schemars(with = "String")]
    pub min_answer: i128,
    #[serde(with = "bignum")]
    #[schemars(with = "String")]
    pub max_answer: i128,

    // Metadata
    pub decimals: u8,
    pub description: String,

    /// Number of faulty oracles the system can tolerate
    pub f: u8,
    /// Total number of oracles
    pub n: u8,

    // Config digest related state
    pub config_count: u32,
    pub latest_config_digest: [u8; 32],
    pub latest_config_block_number: u64,

    // Latest round data
    pub latest_aggregator_round_id: u32,
    pub epoch: u32,
    pub round: u8,

    // Billing fields
    pub billing: Billing,

    pub validator: Option<Validator>,
} // TODO: group some of these into sub-structs

#[allow(clippy::too_many_arguments)]
pub fn config_digest_from_data(
    chain_id: &str,
    contract_address: &Addr,
    config_count: u32,
    signers: &[Vec<u8>],
    transmitters: &[Addr],
    f: u8,
    onchain_config: &[u8],
    offchain_config_version: u64,
    offchain_config: &[u8],
) -> [u8; 32] {
    use blake2::{Blake2s, Digest};
    let mut hasher = Blake2s::default();
    hasher.update(chain_id.as_bytes());
    hasher.update(contract_address.as_bytes());
    hasher.update(&config_count.to_be_bytes());
    hasher.update([(signers.len() as u8)]);
    for signer in signers {
        hasher.update(signer);
    }
    for transmitter in transmitters {
        hasher.update(transmitter.as_bytes());
    }
    hasher.update(&[f]);
    hasher.update((onchain_config.len() as u32).to_be_bytes());
    hasher.update(&onchain_config);
    hasher.update(&offchain_config_version.to_be_bytes());
    hasher.update((offchain_config.len() as u32).to_be_bytes());
    hasher.update(&offchain_config);
    let result = hasher.finalize();
    let mut result: [u8; 32] = result.into();
    // prefix masking
    result[0] = 0x00;
    result[1] = 0x02;
    result
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Transmitter {
    /// Reimbursement in juels
    pub payment: Uint128,
    /// Calculate rewards starting from round id
    pub from_round_id: u32,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Transmission {
    #[serde(with = "bignum")]
    #[schemars(with = "String")]
    pub answer: i128,
    pub observations_timestamp: u32,
    pub transmission_timestamp: u32,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Round {
    pub round_id: u32,
    #[serde(with = "bignum")]
    #[schemars(with = "String")]
    pub answer: i128,
    pub observations_timestamp: u32,
    pub transmission_timestamp: u32,
}

pub const OWNER: Auth = Auth::new("owner");

pub const CONFIG: Item<Config> = Item::new("config");

// An addr currently can't be converted to pubkey: https://docs.cosmos.network/master/architecture/adr-028-public-key-addresses.html

/// index -> sender address
pub const TRANSMITTERS: Map<&Addr, Transmitter> = Map::new("transmitters");
/// index -> ed25519 signing key
pub const SIGNERS: Map<&[u8], ()> = Map::new("signers");

// round ID -> transmission
pub const TRANSMISSIONS: Map<U32Key, Transmission> = Map::new("transmissions");

// Addresses at which oracles want to receive payments.
// transmitter -> payment address
pub const PAYEES: Map<&Addr, Addr> = Map::new("payees");
pub const PROPOSED_PAYEES: Map<&Addr, Addr> = Map::new("proposed_payees");
