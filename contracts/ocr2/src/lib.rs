pub mod contract;
pub mod decimal;
mod error;
mod integration_tests;
pub mod msg;
pub mod state;

pub use crate::decimal::Decimal;
pub use crate::error::ContractError;

pub const fn decimal(i: u64) -> Decimal {
    use cosmwasm_std::Uint128;
    let decimals = 10u128.pow(18);
    let n = i as u128 * decimals;
    Decimal(Uint128::new(n))
}

#[macro_export]
macro_rules! require {
    ($expr:expr, $error:tt) => {
        if !$expr {
            return core::result::Result::Err(ContractError::$error);
        }
    };
}
