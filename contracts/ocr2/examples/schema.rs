use std::env::current_dir;
use std::fs::{create_dir_all, remove_dir_all, rename};

use cosmwasm_schema::{export_schema, remove_schemas, schema_for, write_api};

use ocr2::msg::{
    ExecuteMsg, InstantiateMsg, LatestConfigDetailsResponse, LatestTransmissionDetailsResponse,
    LinkAvailableForPaymentResponse, QueryMsg, TransmittersResponse,
};
use ocr2::state::{Billing, Config, Proposal, Transmitter, Validator};

fn main() {
     // clean directory
     let mut out_dir = current_dir().unwrap();
     out_dir.push("schema");
     remove_dir_all(&out_dir).unwrap();
     create_dir_all(&out_dir).unwrap();

    write_api! {
        instantiate: InstantiateMsg,
        execute: ExecuteMsg,
        query: QueryMsg,
    }


    // put main schema under main folder for codegen.js (else it will error)
    let mut main_dir = out_dir.clone();
    main_dir.push("main");
    create_dir_all(&main_dir).unwrap();

    let mut main_file = out_dir.clone();
    main_file.push("ocr2.json");

    let mut new_location = main_dir.clone();
    new_location.push("ocr2.json");
    rename(&main_file, &new_location).unwrap();


    // put other hand-exported schemas in seperate folder
    let mut other_dir = out_dir.clone();
    other_dir.push("other");
    create_dir_all(&other_dir).unwrap();
    export_schema(&schema_for!(Config), &other_dir);
    export_schema(&schema_for!(Proposal), &other_dir);
    export_schema(&schema_for!(Billing), &other_dir);
    export_schema(&schema_for!(Validator), &other_dir);
    export_schema(&schema_for!(Transmitter), &other_dir);
    export_schema(&schema_for!(LatestConfigDetailsResponse), &other_dir);
    export_schema(&schema_for!(TransmittersResponse), &other_dir);
    export_schema(&schema_for!(LatestTransmissionDetailsResponse), &other_dir);
    export_schema(&schema_for!(LinkAvailableForPaymentResponse), &other_dir);
}
