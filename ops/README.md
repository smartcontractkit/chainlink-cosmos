# Local Testing Environment

Goal: Create a 5 node OCR2 cluster running locally with an active OCR feed for integration testing.

## Using `terrad`
[placeholder for gauntlet]

Need to install `-oracle` version for `terrad`
```bash
git clone https://github.com/terra-money/core/
cd core
git checkout v0.5.7-oracle
make install
```

A key should be attached to `terrad` with the following command.
```bash
terrad keys add <keyname> --recover -i

# example
terrad keys add localterra-deployer --recover -i
```
The name should be added to the corresponding `Pulumi.<type>.yaml` file
```yaml
  terra-env:TERRA-DEPLOYER: localterra-deployer
```

## Using `localterra`

[`LocalTerra`](https://github.com/terra-money/LocalTerra) must be running before running `pulumi up`
```bash
git clone --depth 1 https://www.github.com/terra-money/LocalTerra
cd LocalTerra

# start
docker-compose up

# stop
docker-compose stop

# remove world state
docker-compose rm
```

## Contracts
Unzip the artifacts from https://github.com/smartcontractkit/chainlink-terra/releases and place them under `ops/terrad/artifacts`

Additionally, you'll need the cw20_base from: https://github.com/hackbg/chainlink-terra-cosmwasm-contracts/tree/develop/artifacts

## Pulumi
Infrastructure management tool.

```bash
# create stack for a new network
pulumi stack init <network>

# select network/stack to use
pulumi stack select

# start stack
pulumi up

# stop stack and remove artifacts
pulumi destroy

# remove all traces of stack (usually not needed)
pulumi stack rm <network>
```

Notes:
* Installation: highly recommend using [`asdf`](https://asdf-vm.com/) for version management
   ```
   asdf plugin add pulumi
   asdf install pulumi latest
   asdf global pulumi latest
   ```
* May require setting environment variable `export PULUMI_CONFIG_PASSPHRASE=` (no need to set it to anything unless you want to)
* [Using Pulumi without pulumi.com](https://www.pulumi.com/docs/troubleshooting/faq/#can-i-use-pulumi-without-depending-on-pulumicom): tl;dr - `pulumi login --local`

## Development

In development it could be useful to work on a local copy of `github.com/smartcontractkit/chainlink-relay/ops`, to do so edit `go.mod`:

```
// Notice: pointing to local copy
replace github.com/smartcontractkit/chainlink-relay/ops => ../../chainlink-relay/ops
```
