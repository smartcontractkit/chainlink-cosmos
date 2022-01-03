use cosmwasm_std::{OverflowError, StdError, VerificationError};
use thiserror::Error;

#[derive(Error, Debug, PartialEq)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("{0}")]
    Verification(#[from] VerificationError),

    #[error("{0}")]
    Overflow(#[from] OverflowError),

    #[error("{0}")]
    OwnedError(#[from] owned::Error),

    #[error("Unauthorized")]
    Unauthorized,

    #[error("too many signers")]
    TooManySigners,

    #[error("stale report")]
    StaleReport,

    #[error("config digest mismatch")]
    DigestMismatch,

    #[error("wrong number of signatures")]
    WrongNumberOfSignatures,

    #[error("repeated address")]
    RepeatedAddress,

    #[error("invalid signature")]
    InvalidSignature,

    #[error("invalid input")]
    InvalidInput,

    #[error("payee already set")]
    PayeeAlreadySet,

    #[error("cannot transfer to self")]
    TransferToSelf,

    #[error("median is out of min-max range")]
    MedianOutOfRange,
}
