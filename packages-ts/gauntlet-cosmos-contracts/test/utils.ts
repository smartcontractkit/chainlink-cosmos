import { DirectSecp256k1HdWallet, coins } from '@cosmjs/proto-signing'
import { execSync } from 'child_process'
import { readFileSync, writeFileSync } from 'fs'
import path from 'path'
import UploadCmd from '../src/commands/tooling/upload'
import { SigningCosmWasmClient } from '@cosmjs/cosmwasm-stargate'
import { GasPrice } from '@cosmjs/stargate'
import DeployAC from '../src/commands/contracts/access_controller/deploy'
import DeployLink from '../src/commands/contracts/link/deploy'
import DeployFlags from '../src/commands/contracts/flags/deploy'
import DeployValidator from '../src/commands/contracts/deviation_flagging_validator/deploy'
import DeployOCR2 from '../src/commands/contracts/ocr2/deploy'

export type DeployResponse = {
  responses: {
    tx: string
    contract: string
  }[]
}
export const ONE_TOKEN = BigInt('1000000000000000000')

export const MNEMONIC =
  'surround miss nominee dream gap cross assault thank captain prosper drop duty group candy wealth weather scale put'
export const NODE_URL = 'http://127.0.0.1:26657'
export const DEFAULT_GAS_PRICE = '0.025ucosm'
export const NETWORK = 'local'

export const TIMEOUT = 180000

export const CMD_FLAGS = {
  network: NETWORK,
  mnemonic: MNEMONIC,
  nodeURL: NODE_URL,
  gauntletTest: true,
  defaultGasPrice: DEFAULT_GAS_PRICE,
}

/// Deploy Commands
export const deployOCR2 = async (params: { [key: string]: any }) => {
  const cmd = new DeployOCR2(
    {
      ...CMD_FLAGS,
      ...params,
    },
    [],
  )
  const result = await cmd.run()
  return result['responses'][0]['contract'] as string
}

export const deployAC = async () => {
  const cmd = new DeployAC(
    {
      ...CMD_FLAGS,
    },
    [],
  )
  const result = await cmd.run()
  return result['responses'][0]['contract'] as string
}

export const deployLink = async () => {
  const cmd = new DeployLink(
    {
      ...CMD_FLAGS,
    },
    [],
  )
  await cmd.invokeMiddlewares(cmd, cmd.middlewares)
  const result = ((await cmd.execute()) as unknown) as DeployResponse
  return result.responses[0].contract
}

export const deployFlags = async (raiseAC: string, lowerAC: string) => {
  const cmd = new DeployFlags(
    {
      ...CMD_FLAGS,
      raisingAccessController: raiseAC,
      loweringAccessController: lowerAC,
    },
    [],
  )
  const result = await cmd.run()
  return result['responses'][0]['contract'] as string
}

export const deployValidator = async (flagsAddr: string, threshold: string) => {
  const cmd = new DeployValidator(
    {
      ...CMD_FLAGS,
      flags: flagsAddr,
      flaggingThreshold: threshold,
    },
    [],
  )
  const result = await cmd.run()
  return result['responses'][0]['contract'] as string
}

/// Setup and Teardown Helpers

export const endWasmd = async () => {
  if (process.env.SKIP_WASMD_SETUP) {
    return
  }
  const downScript = path.join(__dirname, '../../../scripts/wasmd.down.sh')
  execSync(`${downScript}`)
}

const WASMD_ACCOUNTS = path.join(__dirname, './devAccounts.json')

/**
 * Initializes Wasmd and Contracts, unless SKIP_WASMD_SETUP=true
 * Can save time for debugging / local testing if Wasmd and base contracts have been setup already
 *
 * Will read initialized account addresses from local file system if SKIP_WASMD_SETUP=true
 *
 * @returns {string[]} Initialized account addresses
 */
export const initWasmd = async () => {
  if (process.env.SKIP_WASMD_SETUP) {
    const rawData = readFileSync(WASMD_ACCOUNTS, 'utf8')
    let { accounts }: { accounts: { address: string; mnemonic: string }[] } = JSON.parse(rawData)
    return await Promise.all(
      accounts.map(async (a) => DirectSecp256k1HdWallet.fromMnemonic(a.mnemonic, { prefix: 'wasm' })),
    )
  }

  const wallets = await startWasmdAndUpload()

  return wallets
}

export const toAddr = async (wallet: DirectSecp256k1HdWallet) => {
  const account = await wallet.getAccounts()
  return account[0].address
}

/**
 * Start Wasmd and Upload Base Contracts.
 *
 * @returns {string[]} Initialized account addresses
 */
export const startWasmdAndUpload = async () => {
  // create test wallets
  let testWallets = await Promise.all(
    Array.from({ length: 4 }, async () => {
      return await DirectSecp256k1HdWallet.generate(12, { prefix: 'wasm' })
    }),
  )
  // add deployer wallet
  const deployerWallet = await DirectSecp256k1HdWallet.fromMnemonic(MNEMONIC, { prefix: 'wasm' })

  testWallets = [deployerWallet, ...testWallets]
  const testAddresses = await Promise.all(testWallets.map(async (wallet) => await toAddr(wallet)))
  const deployerAddress = testAddresses[0]
  const otherAddresses = testAddresses.slice(1)

  const wasmdScript = path.join(__dirname, '../../../scripts/wasmd.sh')

  execSync(wasmdScript)

  // querying wasmd too soon will result in errors
  await new Promise((f) => setTimeout(f, 10000))

  const deployer = await SigningCosmWasmClient.connectWithSigner(NODE_URL, deployerWallet, {
    gasPrice: GasPrice.fromString(DEFAULT_GAS_PRICE),
  })

  // initialize other accounts with some tokens
  for (const testAddr of otherAddresses) {
    await deployer.sendTokens(deployerAddress, testAddr, coins('1', 'ucosm'), 'auto')
  }

  console.log(`All accounts initialized ${testAddresses}`)

  // upload contracts
  process.env.SKIP_PROMPTS = 'true'

  const cmd = new UploadCmd(
    {
      network: NETWORK,
      mnemonic: MNEMONIC,
      nodeURL: NODE_URL,
      gauntletTest: true,
      defaultGasPrice: DEFAULT_GAS_PRICE,
    },
    [],
  )
  await cmd.run()

  // write to local file system for SKIP_WASMD_SETUP=true
  writeFileSync(
    WASMD_ACCOUNTS,
    JSON.stringify({
      accounts: testWallets.map((w, i) => ({ address: testAddresses[i], mnemonic: w.mnemonic })),
    }),
  )

  return testWallets
}
