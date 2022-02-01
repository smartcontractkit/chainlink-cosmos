use std::env::current_dir;
use std::fs::create_dir_all;

use cosmwasm_schema::{export_schema, remove_schemas, schema_for};

use ocr2::msg::{
    ExecuteMsg, InstantiateMsg, LatestConfigDetailsResponse, LatestTransmissionDetailsResponse,
    LinkAvailableForPaymentResponse, QueryMsg, TransmittersResponse,
};
use ocr2::state::{Billing, Config, Transmitter, Validator};

fn main() {
    let mut out_dir = current_dir().unwrap();
    out_dir.push("schema");
    create_dir_all(&out_dir).unwrap();
    remove_schemas(&out_dir).unwrap();

    export_schema(&schema_for!(InstantiateMsg), &out_dir);
    export_schema(&schema_for!(ExecuteMsg), &out_dir);
    export_schema(&schema_for!(QueryMsg), &out_dir);
    export_schema(&schema_for!(Config), &out_dir);
    export_schema(&schema_for!(Billing), &out_dir);
    export_schema(&schema_for!(Validator), &out_dir);
    export_schema(&schema_for!(Transmitter), &out_dir);
    export_schema(&schema_for!(LatestConfigDetailsResponse), &out_dir);
    export_schema(&schema_for!(TransmittersResponse), &out_dir);
    export_schema(&schema_for!(LatestTransmissionDetailsResponse), &out_dir);
    export_schema(&schema_for!(LinkAvailableForPaymentResponse), &out_dir);
}
