import { CosmWasmClient } from '@cosmjs/cosmwasm-stargate'
import { FlagsQueryClient } from '../../codegen/Flags.client'
import { endWasmd, initWasmd, NODE_URL, TIMEOUT, toAddr, deployFlags } from '../utils'

describe('Flags', () => {
  let Flags: FlagsQueryClient
  let flagsAddr: string
  let deployerAddr: string
  let mockRaiseACAddr: string
  let mockLowerACAddr: string
  let aliceAddr: string

  afterAll(async () => {
    await endWasmd()
  })

  beforeAll(async () => {
    // Ideally, we'd start wasmd beforEach() but it takes too long
    const [deployer, mockRaiseAC, mockLowerAC, alice] = await initWasmd()
    deployerAddr = await toAddr(deployer)
    mockRaiseACAddr = await toAddr(mockRaiseAC)
    mockLowerACAddr = await toAddr(mockLowerAC)
    aliceAddr = await toAddr(alice)
    // just give two non-contract addresses
    flagsAddr = await deployFlags(mockRaiseACAddr, mockLowerACAddr)

    const cosmClient = await CosmWasmClient.connect(NODE_URL)
    Flags = new FlagsQueryClient(cosmClient, flagsAddr)
  }, TIMEOUT)

  it(
    'Deploys',
    async () => {
      const owner = await Flags.owner()
      expect(owner).toBe(deployerAddr)

      expect(await Flags.raisingAccessController()).toBe(mockRaiseACAddr)

      // flag is unset for random address
      expect(await Flags.flag({ subject: aliceAddr })).toBe(false)
    },
    TIMEOUT,
  )
})
