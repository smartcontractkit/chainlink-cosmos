#![cfg(test)]
#![cfg(not(tarpaulin_include))]
use crate::contract::{execute, instantiate, query};
use crate::msg::{ExecuteMsg, InstantiateMsg, QueryMsg};
use cosmwasm_std::{attr, Addr, Empty};
use cw_multi_test::{App, AppBuilder, Contract, ContractWrapper, Executor};

use access_controller::msg::ExecuteMsg as AccessControllerMsg;

fn mock_app() -> App {
    AppBuilder::new().build()
}

pub fn contract_flags() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(execute, instantiate, query);
    Box::new(contract)
}

pub fn contract_access_controller() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(
        access_controller::contract::execute,
        access_controller::contract::instantiate,
        access_controller::contract::query,
    );
    Box::new(contract)
}

struct Env {
    router: App,
    owner: Addr,
    raising_access_controller_addr: Addr,
    lowering_access_controller_addr: Addr,
    flags_addr: Addr,
}

fn setup() -> Env {
    let mut router = mock_app();

    let owner = Addr::unchecked("owner");

    let flags_id = router.store_code(contract_flags());
    let access_controller_id = router.store_code(contract_access_controller());

    let raising_access_controller_addr = router
        .instantiate_contract(
            access_controller_id,
            owner.clone(),
            &access_controller::msg::InstantiateMsg {},
            &[],
            "raising_access_controller",
            None,
        )
        .unwrap();

    let lowering_access_controller_addr = router
        .instantiate_contract(
            access_controller_id,
            owner.clone(),
            &access_controller::msg::InstantiateMsg {},
            &[],
            "lowering_access_controller",
            None,
        )
        .unwrap();

    let flags_addr = router
        .instantiate_contract(
            flags_id,
            owner.clone(),
            &InstantiateMsg {
                raising_access_controller: raising_access_controller_addr.to_string(),
                lowering_access_controller: lowering_access_controller_addr.to_string(),
            },
            &[],
            "flags",
            None,
        )
        .unwrap();

    Env {
        router,
        owner,
        raising_access_controller_addr,
        lowering_access_controller_addr,
        flags_addr,
    }
}

#[test]
fn proper_initialization() {
    setup();
}

#[test]
fn raise_flag() {
    let mut env = setup();

    let sender = Addr::unchecked("human");

    // give access to sender
    env.router
        .execute_contract(
            env.owner.clone(),
            env.raising_access_controller_addr.clone(),
            &AccessControllerMsg::AddAccess {
                address: sender.to_string(),
            },
            &[],
        )
        .unwrap();

    let msg = ExecuteMsg::RaiseFlag {
        subject: sender.to_string(),
    };

    env.router
        .execute_contract(sender.clone(), env.flags_addr.clone(), &msg, &[])
        .unwrap();

    let flag: bool = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.flags_addr,
            &QueryMsg::Flag {
                subject: sender.to_string(),
            },
        )
        .unwrap();
    assert_eq!(true, flag);

    // trying to raise the flag when it's already raised
    let msg = ExecuteMsg::RaiseFlag {
        subject: sender.to_string(),
    };

    let res = env
        .router
        .execute_contract(sender.clone(), env.flags_addr.clone(), &msg, &[])
        .unwrap();
    assert_eq!(
        vec![
            attr("action", "already raised flag"),
            attr("subject", sender.clone())
        ],
        res.custom_attrs(1)
    );
}

#[test]
fn raise_flags() {
    let mut env = setup();

    let sender = Addr::unchecked("human");

    // give access to sender
    env.router
        .execute_contract(
            env.owner.clone(),
            env.raising_access_controller_addr.clone(),
            &AccessControllerMsg::AddAccess {
                address: sender.to_string(),
            },
            &[],
        )
        .unwrap();

    let msg = ExecuteMsg::RaiseFlags {
        subjects: vec![sender.to_string()],
    };
    env.router
        .execute_contract(sender.clone(), env.flags_addr.clone(), &msg, &[])
        .unwrap();

    let flags: Vec<bool> = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.flags_addr,
            &QueryMsg::Flags {
                subjects: vec![sender.to_string()],
            },
        )
        .unwrap();

    assert_eq!(vec![true], flags);

    let msg = ExecuteMsg::RaiseFlags {
        subjects: vec![sender.to_string()],
    };
    let res = env
        .router
        .execute_contract(sender.clone(), env.flags_addr.clone(), &msg, &[])
        .unwrap();

    assert_eq!(
        vec![
            attr("action", "already raised flag"),
            attr("subject", sender.clone())
        ],
        res.custom_attrs(1)
    );
}

#[test]
fn lower_flags() {
    let mut env = setup();

    let sender = Addr::unchecked("human");

    // give access to sender
    env.router
        .execute_contract(
            env.owner.clone(),
            env.raising_access_controller_addr.clone(),
            &AccessControllerMsg::AddAccess {
                address: sender.to_string(),
            },
            &[],
        )
        .unwrap();

    let msg = ExecuteMsg::RaiseFlags {
        subjects: vec![sender.to_string()],
    };
    env.router
        .execute_contract(sender.clone(), env.flags_addr.clone(), &msg, &[])
        .unwrap();

    // sender can't lower flags
    let msg = ExecuteMsg::LowerFlags {
        subjects: vec![sender.to_string()],
    };
    let res = env
        .router
        .execute_contract(sender.clone(), env.flags_addr.clone(), &msg, &[]);
    assert!(res.is_err());

    // owner can
    let msg = ExecuteMsg::LowerFlags {
        subjects: vec![sender.to_string()],
    };
    env.router
        .execute_contract(env.owner.clone(), env.flags_addr.clone(), &msg, &[])
        .unwrap();

    // flag is lowered
    let flag: bool = env
        .router
        .wrap()
        .query_wasm_smart(
            &env.flags_addr,
            &QueryMsg::Flag {
                subject: sender.to_string(),
            },
        )
        .unwrap();
    assert_eq!(false, flag);
}
