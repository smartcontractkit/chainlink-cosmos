import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing'
import { execSync } from 'child_process'
import { readFileSync, writeFileSync } from 'fs'
import path from 'path'

import UploadCmd from '../src/commands/tooling/upload'

const MNEMONIC =
  'surround miss nominee dream gap cross assault thank captain prosper drop duty group candy wealth weather scale put'
const NODE_URL = 'http://127.0.0.1:26657'
const DEFAULT_GAS_PRICE = '0.025ucosm'
const NETWORK = 'local'

const TIMEOUT = 90000

const WASMD_ACCOUNTS = path.join(__dirname, './devAccounts.json')

export const CMD_FLAGS = {
  network: NETWORK,
  mnemonic: MNEMONIC,
  nodeURL: NODE_URL,
  gauntletTest: true,
  defaultGasPrice: DEFAULT_GAS_PRICE,
}

export const endWasmd = async () => {
  if (process.env.SKIP_WASMD_SETUP) {
    return
  }
  const downScript = path.join(__dirname, '../../../scripts/wasmd.down.sh')
  execSync(`${downScript}`)
}

/**
 * Initializes Wasmd and Contracts, unless SKIP_WASMD_SETUP=true
 * Can save time for debugging / local testing if Wasmd and base contracts have been setup already
 *
 * Will read initialized account addresses from local file system if SKIP_WASMD_SETUP=true
 *
 * @returns {string[]} Initialized account addresses
 */
export const maybeInitWasmd = async () => {
  const wasmdAccountsPath = path.join(__dirname, './devAccounts.json')

  if (process.env.SKIP_WASMD_SETUP) {
    const rawData = readFileSync(wasmdAccountsPath, 'utf8')
    let data: { accounts: string[] } = JSON.parse(rawData)
    return data.accounts
  }

  const accountAddresses = await startWasmdAndUpload()

  return accountAddresses
}

/**
 * Start Wasmd and Upload Base Contracts.
 *
 * @returns {string[]} Initialized account addresses
 */
export const startWasmdAndUpload = async () => {
  // create other accounts for testing purposes
  const otherAccounts = Array.from({ length: 4 }, async () => {
    const wallet = await DirectSecp256k1HdWallet.generate(12, { prefix: 'wasm' })
    const account = await wallet.getAccounts()
    return account[0]
  })
  let accounts = await Promise.all(otherAccounts)

  const otherAddresses = accounts.map((a) => a.address).join(' ')
  const wasmdScript = path.join(__dirname, '../../../scripts/wasmd.sh')

  execSync(`${wasmdScript} ${otherAddresses}`)

  // retrieve default deployer account
  const deployerAccount = await (async () => {
    const deployerWallet = await DirectSecp256k1HdWallet.fromMnemonic(MNEMONIC, { prefix: 'wasm' })
    const deployerAccounts = await deployerWallet.getAccounts()
    return deployerAccounts[0]
  })()
  accounts = [deployerAccount, ...accounts]

  console.log(`All accounts initialized ${deployerAccount.address} ${otherAddresses}`)

  // querying wasmd too soon will result in errors
  await new Promise((f) => setTimeout(f, 10000))

  // upload contracts
  process.env.SKIP_PROMPTS = 'true'

  const cmd = new UploadCmd(
    {
      network: NETWORK,
      mnemonic: MNEMONIC,
      nodeURL: NODE_URL,
      gauntletTest: true,
      defaultGasPrice: '0.025ucosm',
    },
    [],
  )
  await cmd.run()

  const allAddresses = accounts.map((a) => a.address)

  // write to local file system for SKIP_WASMD_SETUP=true
  writeFileSync(WASMD_ACCOUNTS, JSON.stringify({ accounts: allAddresses }))

  return allAddresses
}
