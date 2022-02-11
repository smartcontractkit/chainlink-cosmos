# Gauntlet Terra Contracts



## Getting started

```
asdf install
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

- Deploy chainlink contract
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

## Multisig Commands

- Create a cw4 group

This instantiates a group with 3 members, each having equal voting power, for a total of 1+1+1=3 votes:  (passing admin address is optional)

```
yarn gauntlet cw4_group:deploy --members='[terra1pl4k5rj2jv6phm2vvhkttju7px6va2ja2v3haw,terra1tsxn3zzp09kvwpx03gzwquhc6nn794vvznuhzr,terra1s66cck3sxacdc2jfpdd4t4pk4yzc60pa72ssdr]' --admin=terra1pl4k5rj2jv6phm2vvhkttju7px6va2ja2v3haw  --network=bombay-testnet
```
- Create a cw3 flex multisig wallet

This instantiates a multisig wallet, for the group above... with a max voting period of 28800s (8 hours) and an threshold of 100 percent of the vote:

```
yarn gauntlet cw3_flex_multisig:deploy --network=bombay-testnet --group=terra1wx0ahe6gpeyyh0wtq3cyc26f2wyk08kjtndxhf --time=28800 --threshold=3
```

You may also specify the threshold as an absolute number of votes, or even a quorum combined with a threshold percentage.
( See https://docs.cosmwasm.com/cw-plus/0.9.0/cw3/cw3-flex-spec for details )

