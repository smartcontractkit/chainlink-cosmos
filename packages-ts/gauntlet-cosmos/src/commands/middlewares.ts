import { Middleware, Next } from '@chainlink/gauntlet-core'
import { assertions, io, logger } from '@chainlink/gauntlet-core/dist/utils'
import { SigningCosmWasmClient } from '@cosmjs/cosmwasm-stargate'
import CosmosCommand from './internal/cosmos'
import path from 'path'
import { existsSync } from 'fs'
import { DirectSecp256k1HdWallet, OfflineSigner } from '@cosmjs/proto-signing'
import { InjectiveDirectEthSecp256k1Wallet, PrivateKey } from '@injectivelabs/sdk-ts'
import { LedgerSigner } from '@cosmjs/ledger-amino'
import TransportNodeHid from '@ledgerhq/hw-transport-node-hid'
import { makeCosmoshubPath } from '@cosmjs/proto-signing'
import { GasPrice } from '@cosmjs/stargate'
import { InjectiveSigningStargateClient } from '@injectivelabs/sdk-ts/dist/cjs/core/stargate'
import { InjectiveClient } from './injective'

const isValidURL = (a) => true

export const withProvider: Middleware = async (c: CosmosCommand, next: Next) => {
  const nodeURL = process.env.NODE_URL
  assertions.assert(
    nodeURL && isValidURL(nodeURL),
    `Invalid NODE_URL (${nodeURL}), please add an http:// or https:// prefix`,
  )

  let wallet: OfflineSigner
  if (c.flags.withLedger || !!process.env.WITH_LEDGER) {
    // TODO: allow specifying custom path, using stringToPath. BIP44_ATOM_PATH was different for example
    // const rawPath = c.flags.ledgerPath || BIP44_ATOM_PATH
    const transport = await TransportNodeHid.create()

    const accounts = [0] // we only use the first account?
    const paths = accounts.map((account) => makeCosmoshubPath(account))

    wallet = new LedgerSigner(transport, {
      // testModeAllowed: true,
      hdPaths: paths,
    })
  } else {
    const mnemonic = process.env.MNEMONIC
    assertions.assert(!!mnemonic, `Missing MNEMONIC, please add one`)

    // wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, { prefix: 'wasm' }) // TODO customizable, in sync with Addr

    const privateKey = PrivateKey.fromMnemonic(mnemonic)
    wallet = (await InjectiveDirectEthSecp256k1Wallet.fromKey(
      Buffer.from(privateKey.toPrivateKeyHex().slice(2), 'hex'),
    )) as OfflineSigner
    // TODO: set hdPaths too, if using different path
  }
  let [signer] = await wallet.getAccounts()

  c.wallet = wallet
  c.signer = signer

  logger.info(`Operator address is ${c.signer.address}`)

  // c.provider = await SigningCosmWasmClient.connectWithSigner(nodeURL, wallet, {
  //   gasPrice: GasPrice.fromString(process.env.DEFAULT_GAS_PRICE),
  // })
  c.provider = await InjectiveClient.connectWithSigner(nodeURL, wallet)
  return next()
}

export const withNetwork: Middleware = (c: CosmosCommand, next: Next) => {
  assertions.assert(
    !!c.flags.network,
    `Network required. Invalid network (${c.flags.network}), please specify a --network`,
  )
  return next()
}

export const withCodeIds: Middleware = (c: CosmosCommand, next: Next) => {
  assertions.assert(
    !!c.flags.network,
    `Network required. Invalid network (${c.flags.network}), please specify a --network`,
  )
  const codeIdsPossiblePaths = [
    path.join(process.cwd(), `./codeIds`),
    path.join(__dirname, `../../../gauntlet-cosmos-contracts/codeIds`),
  ]
  const codeIdPath = codeIdsPossiblePaths
    .filter((codeIdPath) => existsSync(path.join(codeIdPath, `${c.flags.network}.json`)))
    .map((codeIdPath) => path.join(codeIdPath, `${c.flags.network}`))[0]

  const codeIds = io.readJSON(codeIdPath)
  if (!codeIds) logger.error('Code IDs file not found')

  c.codeIds = codeIds || {}
  return next()
}
