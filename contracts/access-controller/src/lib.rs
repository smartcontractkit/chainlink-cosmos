pub mod contract;
mod error;
pub mod msg;
pub mod state;

pub use crate::error::ContractError;

use cosmwasm_std::{Addr, QuerierWrapper, StdResult};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct AccessControllerContract(pub Addr);

impl AccessControllerContract {
    pub fn addr(&self) -> Addr {
        self.0.clone()
    }

    pub fn has_access(&self, querier: &QuerierWrapper, address: &Addr) -> StdResult<bool> {
        state::ACCESS
            .query(querier, self.addr(), address)
            .map(|value| value.is_some())
    }
}

#[macro_export]
macro_rules! require {
    ($expr:expr, $error:tt) => {
        if !$expr {
            return core::result::Result::Err(ContractError::$error);
        }
    };
}
