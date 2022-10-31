pub mod contract;
mod error;
mod integration_tests;
pub mod msg;
pub mod state;

pub use cosmwasm_std::Decimal;

pub use crate::error::ContractError;

#[macro_export]
macro_rules! require {
    ($expr:expr, $error:tt) => {
        if !$expr {
            return core::result::Result::Err(ContractError::$error);
        }
    };
}
