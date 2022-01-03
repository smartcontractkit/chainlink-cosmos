//! Follows the design of `cw-plus/packages/controllers/src/admin.rs`
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};
use thiserror::Error;

use cosmwasm_std::{
    Addr, Deps, DepsMut, Event, MessageInfo, Response, StdError, StdResult, Storage,
};
use cw_storage_plus::Item;

#[derive(Error, Debug, PartialEq)]
pub enum Error {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("Unauthorized")]
    Unauthorized,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct State {
    pub owner: Addr,
    pub proposed_owner: Option<Addr>,
}

pub struct Auth<'a>(Item<'a, State>);

impl<'a> Auth<'a> {
    pub const fn new(namespace: &'a str) -> Self {
        Auth(Item::new(namespace))
    }

    // DepsMut isn't Copy for some reason
    pub fn initialize(&self, storage: &'_ mut dyn Storage, owner: Addr) -> StdResult<()> {
        self.0.save(
            storage,
            &State {
                owner,
                proposed_owner: None,
            },
        )
    }

    pub fn get(&self, deps: Deps) -> StdResult<State> {
        self.0.load(deps.storage)
    }

    /// Returns Ok(true) if this is an owner, Ok(false) if not and an Error if
    /// we hit an error with Api or Storage usage
    pub fn is_owner(&self, deps: Deps, caller: &Addr) -> StdResult<bool> {
        let state = self.0.load(deps.storage)?;
        Ok(caller == &state.owner)
    }

    /// Like is_owner but returns Error::Unauthorized if not owner.
    /// Helper for a nice one-line auth check.
    pub fn assert_owner(&self, deps: Deps, caller: &Addr) -> Result<(), Error> {
        if !self.is_owner(deps, caller)? {
            Err(Error::Unauthorized {})
        } else {
            Ok(())
        }
    }

    pub fn execute_transfer_ownership(
        &self,
        deps: DepsMut,
        info: MessageInfo,
        to: Addr,
    ) -> Result<Response, Error> {
        let mut state = self.0.load(deps.storage)?;

        if info.sender != state.owner {
            return Err(Error::Unauthorized);
        }

        state.proposed_owner = Some(to.clone());

        self.0.save(deps.storage, &state)?;

        Ok(Response::default().add_event(
            Event::new("ownership_transfer_requested")
                .add_attribute("from", state.owner)
                .add_attribute("to", to),
        ))
    }

    pub fn execute_accept_ownership(
        &self,
        deps: DepsMut,
        info: MessageInfo,
    ) -> Result<Response, Error> {
        let mut state = self.0.load(deps.storage)?;

        if Some(&info.sender) != state.proposed_owner.as_ref() {
            return Err(Error::Unauthorized);
        }

        let old_owner = std::mem::replace(&mut state.owner, info.sender);
        state.proposed_owner = None;
        self.0.save(deps.storage, &state)?;

        Ok(Response::default().add_event(
            Event::new("ownership_transfer_requested")
                .add_attribute("from", old_owner)
                .add_attribute("to", state.owner),
        ))
    }

    pub fn query_owner(&self, deps: Deps) -> StdResult<Addr> {
        let state = self.get(deps)?;
        Ok(state.owner)
    }
}

#[cfg(test)]
#[cfg(not(tarpaulin_include))]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{mock_dependencies, mock_info};

    #[test]
    fn initialize_owner() {
        let mut deps = mock_dependencies(&[]);
        let control = Auth::new("foo");

        // initialize and check
        let owner = Addr::unchecked("owner");
        control
            .initialize(&mut deps.storage, owner.clone())
            .unwrap();
        let got = control.get(deps.as_ref()).unwrap();
        assert_eq!(owner, got.owner);
    }

    #[test]
    fn owner_checks() {
        let mut deps = mock_dependencies(&[]);

        let control = Auth::new("foo");
        let owner = Addr::unchecked("big boss");
        let imposter = Addr::unchecked("imposter");

        // ensure checks proper with owner set
        control
            .initialize(&mut deps.storage, owner.clone())
            .unwrap();
        assert!(control.is_owner(deps.as_ref(), &owner).unwrap());
        assert!(!(control.is_owner(deps.as_ref(), &imposter).unwrap()));
        control.assert_owner(deps.as_ref(), &owner).unwrap();
        let err = control.assert_owner(deps.as_ref(), &imposter).unwrap_err();
        assert_eq!(Error::Unauthorized, err);
    }

    #[test]
    fn transfer_accept_ownership() {
        let mut deps = mock_dependencies(&[]);

        // initial setup
        let control = Auth::new("foo");
        let owner = Addr::unchecked("big boss");
        let imposter = Addr::unchecked("imposter");
        let friend = Addr::unchecked("buddy");
        control
            .initialize(deps.as_mut().storage, owner.clone())
            .unwrap();

        // query shows results
        let res = control.query_owner(deps.as_ref()).unwrap();
        assert_eq!(owner, res);

        // imposter cannot initiate transfer
        let info = mock_info(imposter.as_ref(), &[]);
        let new_owner = friend.clone();
        let err = control
            .execute_transfer_ownership(deps.as_mut(), info, new_owner.clone())
            .unwrap_err();
        assert_eq!(Error::Unauthorized, err);

        // owner can initiate transfer
        let info = mock_info(owner.as_ref(), &[]);
        let res = control
            .execute_transfer_ownership(deps.as_mut(), info, new_owner.clone())
            .unwrap();
        assert_eq!(0, res.messages.len());

        // query still shows original owner
        let res = control.query_owner(deps.as_ref()).unwrap();
        assert_eq!(owner, res);

        // imposter cannot accept transfer
        let info = mock_info(imposter.as_ref(), &[]);
        let err = control
            .execute_accept_ownership(deps.as_mut(), info)
            .unwrap_err();
        assert_eq!(Error::Unauthorized, err);

        // proposed owner can accept transfer
        let info = mock_info(new_owner.as_ref(), &[]);
        let res = control
            .execute_accept_ownership(deps.as_mut(), info)
            .unwrap();
        assert_eq!(0, res.messages.len());

        // query now shows new owner
        let res = control.query_owner(deps.as_ref()).unwrap();
        assert_eq!(new_owner, res);
    }
}
