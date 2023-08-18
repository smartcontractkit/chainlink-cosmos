import DeployAC from '../../src/commands/contracts/access_controller/deploy'
import { CosmWasmClient } from '@cosmjs/cosmwasm-stargate'
import { AccessControllerQueryClient } from '../../codegen/AccessController.client'
import { endWasmd, CMD_FLAGS, maybeInitWasmd, NODE_URL, TIMEOUT, ONE_TOKEN, DeployResponse } from '../utils'

const deployAC = async () => {
  const cmd = new DeployAC(
    {
      ...CMD_FLAGS,
    },
    [],
  )
  const result = await cmd.run()
  return result
}

describe('Link', () => {
  let AccessController: AccessControllerQueryClient
  let ACAddr: string
  let deployerAddr: string
  let aliceAddr: string
  let bobAddr: string
  let usersAddr: string[]

  afterAll(async () => {
    await endWasmd()
  })

  beforeAll(async () => {
    // Ideally, we'd start wasmd beforEach() but it takes too long
    [deployerAddr, aliceAddr, bobAddr, ...usersAddr] = await maybeInitWasmd()
    const res = await deployAC()
    ACAddr = res['responses'][0]['contract'] as string
    console.log(ACAddr, 'pinenut')

    const cosmClient = await CosmWasmClient.connect(NODE_URL)
    AccessController = new AccessControllerQueryClient(cosmClient, ACAddr)
  }, TIMEOUT)

  it.only('Deploys', async () => {

    const owner = await AccessController.owner()
    expect(owner).toBe(deployerAddr)
  }, TIMEOUT)

})
