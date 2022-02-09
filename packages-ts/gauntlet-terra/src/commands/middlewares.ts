import { LCDClient, Key, MnemonicKey } from '@terra-money/terra.js'
import { LedgerKey } from './ledgerKey'
import { Middleware, Next } from '@chainlink/gauntlet-core'
import { assertions, io, logger } from '@chainlink/gauntlet-core/dist/utils'
import TerraCommand from './internal/terra'
import path from 'path'
import { existsSync } from 'fs'
import { LEDGER_ULUNA_PATH } from '../lib/constants'

const isValidURL = (a) => true
export const withProvider: Middleware = (c: TerraCommand, next: Next) => {
  const nodeURL = process.env.NODE_URL
  assertions.assert(
    nodeURL && isValidURL(nodeURL),
    `Invalid NODE_URL (${nodeURL}), please add an http:// or https:// prefix`,
  )
  c.provider = new LCDClient({
    URL: nodeURL,
    chainID: process.env.CHAIN_ID,
    gasPrices: { uluna: process.env.DEFAULT_GAS_PRICE },
  })
  return next()
}

export const withWallet: Middleware = async (c: TerraCommand, next: Next) => {
  let key: Key
  if (c.flags.withLedger || !!process.env.WITH_LEDGER) {
    const path = c.flags.ledgerPath || LEDGER_ULUNA_PATH
    key = await LedgerKey.create(path)
  } else {
    const mnemonic = process.env.MNEMONIC
    assertions.assert(!!mnemonic, `Missing MNEMONIC, please add one`)

    key = new MnemonicKey({
      mnemonic: mnemonic,
    })
  }

  c.wallet = c.provider.wallet(key)
  console.info(`Operator address is ${c.wallet.key.accAddress}`)
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
  const codeIdsPossiblePaths = [`./codeIds`, `./packages-ts/gauntlet-terra-contracts/codeIds`]
  const codeIdPath = codeIdsPossiblePaths
    .filter((codeIdPath) => existsSync(path.join(process.cwd(), `${codeIdPath}/${c.flags.network}.json`)))
    .map((codeIdPath) => path.join(process.cwd(), `${codeIdPath}/${c.flags.network}`))[0]

  const codeIds = io.readJSON(codeIdPath)
  if (!codeIds) logger.error('Code IDs file not found')

  c.codeIds = codeIds || {}
  return next()
}
