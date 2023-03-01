# Getting Started With Gauntlet

`gauntlet` is a CLI tool that helps you deploy and interact with smart contracts. Gauntlet for Terra enables you to perform the following tasks.

- Deploy Chainlink smart contracts
- Initialize and fund OCR2 contracts
- Transfer ownership of contracts using multisig
- Update allow list in access controller contracts
- Query any contract on Terra
- Send ATOM
- And more!

## Quickstart

### 1. Clone the Chainlink Terra repository

```bash
git clone https://github.com/smartcontractkit/chainlink-cosmos.git
```

### 2. Install dependencies

Run the following command.

```bash
make install
```

### 3. Configure environment

Create a `.env` file in the root directory of `chainlink-cosmos` and add the following line.

```bash
MNEMONIC=replace with your mnemonic
```

A deterministic Terra wallet key will be generated from the defined mnemonic phrase.

### 4. Run Gauntlet

To check available commands, run the following.

```bash
yarn gauntlet -h
```

To specify a network, use the `--network` flag.

```bash
yarn gauntlet tooling:query --network=testnet-bombay-internal --msg='{"owed_payment":{"transmitter": "terra1myd0kxk3fqaz9zl47gm2uvxjm0zn3lczrtvljz"}}' terra14mf0qcjpduhcs8p6289pjnwn8skhgk5aus3yxg
```

### 5. Configuring a new network

The network name must be selected from the [networks](../packages-ts/gauntlet-terra-contracts/networks/) folder. To add a new network configuration, simply add a new file to this folder (.env.YOUR_NETWORK_NAME).

The following code snippet defines the general structure for defining a new network:

```
NODE_URL=
CHAIN_ID=columbus-5
DEFAULT_GAS_PRICE=0.5
```

The only thing that needs to be updated here is the `NODE_URL`, which can be retrieved from [Terra Public LCD Endpoints](https://docs.terra.money/docs/develop/endpoints.html#public-lcd)
