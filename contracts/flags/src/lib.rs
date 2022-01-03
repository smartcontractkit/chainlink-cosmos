pub mod contract;
pub mod error;
mod integration_tests;
pub mod msg;
pub mod state;

#[macro_export]
macro_rules! require {
    ($expr:expr, $error:tt) => {
        if !$expr {
            return core::result::Result::Err(ContractError::$error);
        }
    };
}
