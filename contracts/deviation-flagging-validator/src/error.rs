use cosmwasm_std::StdError;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("{0}")]
    OwnedError(#[from] owned::Error),

    /// Only callable by owner
    #[error("Only callable by owner")]
    Unauthorized,
}
