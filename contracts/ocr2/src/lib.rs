pub mod contract;
pub mod decimal;
mod error;
mod integration_tests;
pub mod msg;
pub mod state;

pub use crate::decimal::Decimal;
// NOTE: if cosmwasm ever fixes https://github.com/CosmWasm/cosmwasm/issues/1156
// switch back after upgrading to cosmwasm-std 1.0 which also supports (Decimal * Decimal)
// pub use cosmwasm_std::Decimal;
pub use crate::error::ContractError;

#[macro_export]
macro_rules! require {
    ($expr:expr, $error:tt) => {
        if !$expr {
            return core::result::Result::Err(ContractError::$error);
        }
    };
}
