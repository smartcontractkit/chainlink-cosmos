import { CosmWasmClient } from '@cosmjs/cosmwasm-stargate'
import { DeviationFlaggingValidatorQueryClient } from '../../codegen/DeviationFlaggingValidator.client'
import { endWasmd, initWasmd, NODE_URL, TIMEOUT, toAddr, deployFlags, deployValidator } from '../utils'

describe('Deviation Flagging Validator', () => {
  let Validator: DeviationFlaggingValidatorQueryClient
  let validatorAddr: string
  let flagsAddr: string
  let deployerAddr: string

  afterAll(async () => {
    await endWasmd()
  })

  beforeAll(async () => {
    // Ideally, we'd start wasmd beforEach() but it takes too long
    const [deployer, mockRaiseAC, mockLowerAC] = await initWasmd()
    deployerAddr = await toAddr(deployer)
    let mockRaiseACAddr = await toAddr(mockRaiseAC)
    let mockLowerACAddr = await toAddr(mockLowerAC)
    // just give two non-contract addresses
    flagsAddr = await deployFlags(mockRaiseACAddr, mockLowerACAddr)

    // threshold is effectively "1" or "100%"" for simplicity
    validatorAddr = await deployValidator(flagsAddr, '100000')

    const cosmClient = await CosmWasmClient.connect(NODE_URL)
    Validator = new DeviationFlaggingValidatorQueryClient(cosmClient, validatorAddr)
  }, TIMEOUT)

  it(
    'Deploys',
    async () => {
      const owner = await Validator.owner()
      expect(owner).toBe(deployerAddr)

      // deviation of 100% is valid
      expect(await Validator.isValid({ answer: '2', previousAnswer: '1' })).toBe(true)
      // deviation over 100% is not valid
      expect(await Validator.isValid({ answer: '3', previousAnswer: '1' })).toBe(false)

      expect((await Validator.flaggingThreshold()).threshold).toBe(100000)
    },
    TIMEOUT,
  )
})
