# Gauntlet Terra Contracts



## Getting started

```
yarn
yarn build
yarn gauntlet <command> --<flags>=<x> <arguments>
```

To bundle:
```
yarn
yarn bundle
```

This will generate executables for Linux and MacOS under `./bin`. 
```
./bin/gauntlet-terra-<macos/linux> <command>  --<flags>=<x> <arguments>
```
## Commands

Every contract available has 3 actions available:
- Deploy: Deploys the contract
- Execute/Query: Executes/Queries a contract function. Execute will send a transaction. Query will inspect the contract
- Help: Gives an description of the contract available functions

The command follows the same style:
```
<contract>:<action> --<param>=<value> <contract_address>
```

For our Access Controller contract, we could perform:

- Deploy
```
./bin/gauntlet-terra-macos access_controller:deploy --network=bombay-testnet
```
This will give us the contract address (`terra234`)

- To Execute/Query any function
```
./bin/gauntlet-terra-macos access_controller:add_access --address="terra123" terra234
```
Should add access for address `terra123`
```
./bin/gauntlet-terra-macos access_controller:has_access --address="terra123" terra234
```
Should return `true` has `terra123` has already access

- To show available methods:
```
./bin/gauntlet-terra-macos access_controller:help
```
It will show every method available in the contract, with their needed parameters and their types, if any


The following contracts have the previous actions available:
- `access_controller`
- `flags`
- `ocr2`
- `deviation_flagging_validator`