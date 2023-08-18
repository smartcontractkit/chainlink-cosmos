import DeployLinkCmd from '../../src/commands/contracts/link/deploy'
import MintLinkCmd from '../../src/commands/contracts/link/mint'
import TransferLinkCmd from '../../src/commands/contracts/link/transfer'
import { CosmWasmClient } from '@cosmjs/cosmwasm-stargate'
import { CW20BaseQueryClient } from '../../codegen/CW20Base.client'
import { endWasmd, CMD_FLAGS, maybeInitWasmd } from '../utils'

type DeployResponse = {
  responses: {
    tx: string
    contract: string
  }[]
}

const ONE_TOKEN = BigInt('1000000000000000000')
const MNEMONIC =
  'surround miss nominee dream gap cross assault thank captain prosper drop duty group candy wealth weather scale put'
const NODE_URL = 'http://127.0.0.1:26657'
const DEFAULT_GAS_PRICE = '0.025ucosm'
const NETWORK = 'local'
const TIMEOUT = 90000

const deployLink = async () => {
  const cmd = new DeployLinkCmd(
    {
      ...CMD_FLAGS,
    },
    [],
  )
  await cmd.invokeMiddlewares(cmd, cmd.middlewares)
  const result = ((await cmd.execute()) as unknown) as DeployResponse
  return result.responses[0].contract
}

describe('Link', () => {
  let Link: CW20BaseQueryClient
  let linkAddr: string
  let deployerAddr: string
  let aliceAddr: string
  let bobAddr: string
  let usersAddr: string[]

  afterAll(async () => {
    await endWasmd()
  })

  beforeAll(async () => {
    // Ideally, we'd start wasmd beforEach() but it takes too long
    ;[deployerAddr, aliceAddr, bobAddr, ...usersAddr] = await maybeInitWasmd()
    linkAddr = await deployLink()

    const cosmClient = await CosmWasmClient.connect(NODE_URL)
    Link = new CW20BaseQueryClient(cosmClient, linkAddr)
  }, TIMEOUT)

  it(
    'Deploys',
    async () => {
      expect(linkAddr.includes('wasm')).toBe(true)

      const { decimals, name, symbol, total_supply } = await Link.tokenInfo()

      expect(decimals).toEqual(18)
      expect(name).toEqual('ChainLink Token')
      expect(symbol).toEqual('LINK')
      expect(BigInt(total_supply)).toEqual(ONE_TOKEN * BigInt(1_000_000_000))

      const { balance: deployerBalance } = await Link.balance({ address: deployerAddr })
      expect(BigInt(deployerBalance)).toEqual(ONE_TOKEN * BigInt(1_000_000_000))

      const { project, logo, description } = await Link.marketingInfo()

      expect(project).toEqual('Chainlink')
      expect(logo).not.toBeNull()
      expect(description).toBeNull()

      const { minter } = await Link.minter()
      expect(minter).toEqual(deployerAddr)
    },
    TIMEOUT,
  )

  it(
    'Mints To Address',
    async () => {
      const mint = new MintLinkCmd(
        {
          ...CMD_FLAGS,
          to: aliceAddr,
          amount: 1,
        },
        [linkAddr],
      )
      await mint.run()

      const aliceBalance = BigInt((await Link.balance({ address: aliceAddr })).balance)
      expect(aliceBalance).toEqual(ONE_TOKEN)
    },
    TIMEOUT,
  )

  it(
    'Transfers',
    async () => {
      const transfer = new TransferLinkCmd(
        {
          ...CMD_FLAGS,
          to: bobAddr,
          amount: 1,
        },
        [linkAddr],
      )
      await transfer.run()

      const bobBalance = BigInt((await Link.balance({ address: bobAddr })).balance)
      expect(bobBalance).toEqual(ONE_TOKEN)
    },
    TIMEOUT,
  )
})
