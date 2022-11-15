import { Middleware, Next } from '@chainlink/gauntlet-core'
import { assertions, io, logger } from '@chainlink/gauntlet-core/dist/utils'
import { SigningCosmWasmClient } from '@cosmjs/cosmwasm-stargate'
import TerraCommand from './internal/terra'
import path from 'path'
import { existsSync } from 'fs'
import { BIP44_LUNA_PATH } from '../lib/constants'
import { DirectSecp256k1HdWallet, OfflineSigner } from '@cosmjs/proto-signing'
import { LedgerSigner } from "@cosmjs/ledger-amino";
import TransportNodeHid from '@ledgerhq/hw-transport-node-hid'
import { makeCosmoshubPath } from '@cosmjs/proto-signing';

const isValidURL = (a) => true

export const withProvider: Middleware = async (c: TerraCommand, next: Next) => {
  const nodeURL = process.env.NODE_URL
  assertions.assert(
    nodeURL && isValidURL(nodeURL),
    `Invalid NODE_URL (${nodeURL}), please add an http:// or https:// prefix`,
  )

  let wallet: OfflineSigner
  if (c.flags.withLedger || !!process.env.WITH_LEDGER) {
    const rawPath = c.flags.ledgerPath || BIP44_LUNA_PATH
    const transport = await TransportNodeHid.create()
    const BIP44_REGEX = /^(44)\'\s*\/\s*(\d+)\'\s*\/\s*([0,1]+)\'\s*\/\s*(\d+)\s*\/\s*(\d+)$/
    const match = BIP44_REGEX.exec(rawPath)
    if (!match) throw new Error('Invalid BIP44 path!')
    const path = match.slice(1).map((i) => makeCosmoshubPath(Number(i)))

    wallet = new LedgerSigner(transport, {
      // testModeAllowed: true,
      hdPaths: path,
    });    
  } else {
    const mnemonic = process.env.MNEMONIC
    assertions.assert(!!mnemonic, `Missing MNEMONIC, please add one`)
    wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic);
  }
  let [signer] = await wallet.getAccounts();

  logger.debug(`Operator address is ${c.signer.address}`)
  
  c.wallet = wallet;
  c.signer = signer;

  c.provider = await SigningCosmWasmClient.connectWithSigner(nodeURL, wallet)
  // c.oldProvider = new LCDClient({
  //   URL: nodeURL,
  //   chainID: process.env.CHAIN_ID,
  //   gasPrices: { ucosm: process.env.DEFAULT_GAS_PRICE },
  // })
  return next()
}

export const withNetwork: Middleware = (c: TerraCommand, next: Next) => {
  assertions.assert(
    !!c.flags.network,
    `Network required. Invalid network (${c.flags.network}), please specify a --network`,
  )
  return next()
}

export const withCodeIds: Middleware = (c: TerraCommand, next: Next) => {
  assertions.assert(
    !!c.flags.network,
    `Network required. Invalid network (${c.flags.network}), please specify a --network`,
  )
  const codeIdsPossiblePaths = [
    path.join(process.cwd(), `./codeIds`),
    path.join(__dirname, `../../../gauntlet-terra-contracts/codeIds`),
  ]
  const codeIdPath = codeIdsPossiblePaths
    .filter((codeIdPath) => existsSync(path.join(codeIdPath, `${c.flags.network}.json`)))
    .map((codeIdPath) => path.join(codeIdPath, `${c.flags.network}`))[0]

  const codeIds = io.readJSON(codeIdPath)
  if (!codeIds) logger.error('Code IDs file not found')

  c.codeIds = codeIds || {}
  return next()
}
