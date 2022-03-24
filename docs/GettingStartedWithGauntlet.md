# Getting Started With Gauntlet

`gauntlet` is a CLI tool that helps you deploy and interact with smart contracts. Gauntlet for Terra enables you to perform the following tasks.

- Deploy Chainlink smart contracts
- Initialize and fund OCR2 contracts
- Transfer ownership of contracts using multisig
- Update allow list in access controller contracts
- Query any contract on Terra
- Send Luna
- And more!

## Quickstart

### 1. Clone the Chainlink Terra repository

```bash
git clone https://github.com/smartcontractkit/chainlink-terra.git
```

### 2. Set up `nvm` and install dependencies

Install [Node Version Manager](https://github.com/nvm-sh/nvm) to help you manage multiple Node.js versions if you haven't already. Then, run the following commands.

```bash
cd chainlink-terra
nvm use
yarn
```

### 3. Configure environment

Create a `.env` file in the root directory of `chainlink-terra` and add the following line.

```bash
MNEMONIC=replace with your mnemonic
```

MNEMONIC must be set equal to the mnemonic associated with your Terra wallet.

### 4. Run Gauntlet

To check available commands, run the following.

```bash
yarn gauntlet -h
```

To specify a network, use the `--network` flag.

```bash
yarn gauntlet tooling:query --network=testnet-bombay-internal --msg='{"owed_payment":{"transmitter": "terra1myd0kxk3fqaz9zl47gm2uvxjm0zn3lczrtvljz"}}' terra14mf0qcjpduhcs8p6289pjnwn8skhgk5aus3yxg
```

The network name must be selected from the [networks](../packages-ts/gauntlet-terra-contracts/networks/) folder. To add a new network configuration, simply add a new file to this folder (.env.YOUR_NETWORK_NAME).
