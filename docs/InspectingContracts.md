# Inspecting Terra Contracts

Contract schemas for Terra can be found in the [contracts](/contracts) folder. Each contract in this directory has a subdirectory called **schema**. In these folders, you can learn how to query, execute, and instantiate each contract.

## Queries

Queries are located in the **query_msg.json** files.

There are two types of queries: string queries and object queries. String queries are used to read a property of the contract, such as the contract owner or the billing access controller. The [OCR2 contract](/contracts/ocr2/schema/query_msg.json) outlines a few possible string queries.

```json
{
    "type": "string",
    "enum": [
    "latest_config_details",
    "transmitters",
    "latest_transmission_details",
    "latest_config_digest_and_epoch",
    "description",
    "decimals",
    "latest_round_data",
    "link_token",
    "billing",
    "billing_access_controller",
    "requester_access_controller",
    "link_available_for_payment",
    "version",
    "owner"
    ]
}
```

Object queries are used when the query is dependent upon a variable input. The following is the JSON representation of a possible object query to the [OCR2 contract](/contracts/ocr2/schema/query_msg.json).

```json
{
    "type": "object",
    "required": [
        "owed_payment"
    ],
    "properties": {
        "owed_payment": {
            "type": "object",
            "required": [
                "transmitter"
            ],
            "properties": {
                "transmitter": {
                    "type": "string"
                }
            }
        }
    },
    "additionalProperties": false
}
```

This representation indicates that the query is an object that requires another object called "owed_payment" that requires a string called "transmitter". The JSON query would be of the following format.

```json
{
    "owed_payment": {
        "transmitter": "TRANSMITTER_ADDRESS"
    }
}
```

### Queries Via Terra Finder

On the [Terra Finder](https://finder.terra.money/) home page, search the address of the contract that you want to inspect. For example, to inspect the OCR2 contract for the USD/USDT feed on testnet, search *terra14mf0qcjpduhcs8p6289pjnwn8skhgk5aus3yxg*. Make sure you switch to testnet in the top right hand corner of the home page before you search.

Once on the smart contract page, you can create a JSON query. Below is a valid JSON query to the proxy contract, where *terra1myd0kxk3fqaz9zl47gm2uvxjm0zn3lczrtvljz* is an address that has performed a transaction with the contract. Transactions can also be viewed in the smart contract page.

```json
{
    "owed_payment": {
        "transmitter": "terra1myd0kxk3fqaz9zl47gm2uvxjm0zn3lczrtvljz"
    }
}
```

You can also perform string queries if you use escape characters.

```json
"\"owner\""
```

### Queries Via Terrad

Terrad is a CLI tool that enables users to interact with the Terra blockchain. To query a contract, you must use *terrad query wasm contract-store*.

```bash
Usage:
  terrad query wasm contract-store [bech32-address] [msg] [flags]

Flags:
      --height int      Use a specific height to query state at (this can error if the node is pruning state)
  -h, --help            help for contract-store
      --node string     <host>:<port> to Tendermint RPC interface for this chain (default "tcp://localhost:26657")
  -o, --output string   Output format (text|json) (default "text")

Global Flags:
      --chain-id string     The network chain ID
      --home string         directory for config and data (default "/Users/kylemartin/.terra")
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic) (default "info")
      --trace               print out full stack trace on errors
```

Below is an example query to the OCR2 contract. 

```bash
terrad query wasm contract-store terra14mf0qcjpduhcs8p6289pjnwn8skhgk5aus3yxg '{"owed_payment":{"transmitter": "terra1myd0kxk3fqaz9zl47gm2uvxjm0zn3lczrtvljz"}}' --node "https://RPC_URL:443"
```

The response is as follows.

```bash
query_result: "3013470097000000000"
```

### Queries Via cURL

You can also interact with deployed contracts via cURL. Queries using cURL will have the following format, where *query* is either a string or a URL encoded JSON object.

```bash
curl '${fcd_endpoint}/wasm/contracts/${address}/store?query_msg=${query}'
```

Some examples of how to query the USD/USDT proxy contract can be found below.

```bash
curl 'https://bombay-fcd.terra.dev/wasm/contracts/terra134lgzqfwms0sg4a33wpygj8waff2d704gcezsu/store?query_msg="owner"'
curl 'https://bombay-fcd.terra.dev/wasm/contracts/terra134lgzqfwms0sg4a33wpygj8waff2d704gcezsu/store?query_msg=%7B%22round_data%22:%7B%22round_id%22:4294968601%7D%7D'
```

Note that *%7B%22round_data%22:%7B%22round_id%22:4294968601%7D%7D* is the URL encoded format of *{"round_data":{"round_id":4294968601}}*. The sample responses for each of the cURL commands above are as follows.

```bash
# curl 'https://bombay-fcd.terra.dev/wasm/contracts/terra134lgzqfwms0sg4a33wpygj8waff2d704gcezsu/store?query_msg="owner"'
{"height":"8260930","result":"terra19mz966zzv34tr7vxu0z66ps2ey20mv3nfdzukd"}
# curl 'https://bombay-fcd.terra.dev/wasm/contracts/terra134lgzqfwms0sg4a33wpygj8waff2d704gcezsu/store?query_msg=%7B%22round_data%22:%7B%22round_id%22:4294968601%7D%7D'
{"height":"8260933","result":{"round_id":4294968601,"answer":"521361092112","observations_timestamp":1646914654,"transmission_timestamp":1646914668}}
```

### Queries Via Gauntlet

You can also use [Gauntlet Terra](../packages-ts/gauntlet-cosmos-contracts/) to query smart contracts. To query a smart contract using Gauntlet, run the following command:

```bash
yarn gauntlet tooling:query --network=[NETWORK_NAME] --msg=[QUERY] [CONTRACT_ADDRESS]
yarn gauntlet tooling:query --network=testnet-bombay-internal --msg='{"owed_payment":{"transmitter": "terra1myd0kxk3fqaz9zl47gm2uvxjm0zn3lczrtvljz"}}' terra14mf0qcjpduhcs8p6289pjnwn8skhgk5aus3yxg
```

The response of the above command can be found below.

```bash
yarn run v1.22.17
$ yarn build && node ./packages-ts/gauntlet-cosmos-contracts/dist/index.js query --network=testnet-bombay-internal '--msg={"owed_payment":{"transmitter": "terra1myd0kxk3fqaz9zl47gm2uvxjm0zn3lczrtvljz"}}' terra14mf0qcjpduhcs8p6289pjnwn8skhgk5aus3yxg
$ yarn clean && tsc -b ./tsconfig.json
$ tsc -b --clean ./tsconfig.json
🧤  gauntlet 0.0.7
ℹ️   Query finished with result: "3515090726000000000"
✨  Done in 9.59s.
```

### Gauntlet Inspect
