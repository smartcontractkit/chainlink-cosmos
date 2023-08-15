use cosmwasm_std::Addr;
use cosmwasm_schema::{cw_serde, QueryResponses};

// TODO: Deduplicate (also declared in 'contracts/ocr2/src/state.rs')
// https://github.com/smartcontractkit/chainlink-cosmos/issues/18
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

#[cw_serde]
#[derive(Eq)]
pub struct InstantiateMsg {
    /// The address of the flags contract
    pub flags: String,
    /// The threshold that will trigger a flag to be raised
    /// Setting the value of 100,000 is equivalent to tolerating a 100% change
    /// compared to the previous price
    pub flagging_threshold: u32,
}

#[cw_serde]
#[derive(Eq)]
pub enum ExecuteMsg {
    /// Initiate contract ownership transfer to another address.
    /// Can be used only by owner
    TransferOwnership {
        /// Address to transfer ownership to
        to: String,
    },
    /// Finish contract ownership transfer. Can be used only by pending owner
    AcceptOwnership,
    /// Updates the flagging threshold
    /// Can be used only by owner
    SetFlaggingThreshold { threshold: u32 },
    /// Updates the flagging contract address for raising flags
    /// Can be used only by owner
    SetFlagsAddress { flags: Addr },
    /// Checks whether the parameters count as valid by comparing the difference
    /// change to the flagging threshold
    Validate {
        /// ID of the previous round
        previous_round_id: u32,
        /// Previous answer, used as the median of difference with the current
        /// answer to determine if the deviation threshold has been exceeded
        #[serde(with = "bignum")]
        #[schemars(with = "String")]
        previous_answer: i128,
        /// ID of the current round
        round_id: u32,
        /// Current answer which is compared for a ration of change to make sure
        /// it has not exceeded the flagging threshold
        #[serde(with = "bignum")]
        #[schemars(with = "String")]
        answer: i128,
    },
}

#[cw_serde]
#[derive(Eq, QueryResponses)]
pub enum QueryMsg {
    /// Check whether the parameters count is valid by comparing the difference
    /// change to the flagging threshold
    /// Res
    #[returns(bool)]
    IsValid {
        /// Previous answer, used as the median of difference with the current
        /// answer to determine if the deviation threshold has been exceeded
        #[serde(with = "bignum")]
        #[schemars(with = "String")]
        previous_answer: i128,
        /// Current answer which is compared for a ration of change to make sure
        /// it has not exceeded the flagging threshold
        #[serde(with = "bignum")]
        #[schemars(with = "String")]
        answer: i128,
    },
    /// Query the flagging threshold
    /// Response: [`FlaggingThresholdResponse`]
    #[returns(FlaggingThresholdResponse)]
    FlaggingThreshold,
    /// Returns contract owner's address
    /// Response [`Addr`]
    #[returns(Addr)]
    Owner,
}

#[cw_serde]
#[derive(Eq)]
pub struct FlaggingThresholdResponse {
    pub threshold: u32,
}
