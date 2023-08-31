import AddAccess from '../../src/commands/contracts/access_controller/addAccess'
import RemoveAccess from '../../src/commands/contracts/access_controller/removeAccess'
import { CosmWasmClient } from '@cosmjs/cosmwasm-stargate'
import { AccessControllerQueryClient } from '../../codegen/AccessController.client'
import { endWasmd, CMD_FLAGS, initWasmd, toAddr, NODE_URL, TIMEOUT, deployAC } from '../utils'

describe('Access Controller', () => {
  let AccessController: AccessControllerQueryClient
  let ACAddr: string
  let deployerAddr: string
  let aliceAddr: string

  afterAll(async () => {
    await endWasmd()
  })

  beforeAll(async () => {
    // Ideally, we'd start wasmd beforEach() but it takes too long
    const [deployer, alice] = await initWasmd()
    deployerAddr = await toAddr(deployer)
    aliceAddr = await toAddr(alice)
    ACAddr = await deployAC()

    const cosmClient = await CosmWasmClient.connect(NODE_URL)
    AccessController = new AccessControllerQueryClient(cosmClient, ACAddr)
  }, TIMEOUT)

  it(
    'Deploys',
    async () => {
      const owner = await AccessController.owner()
      expect(owner).toBe(deployerAddr)
      expect(await AccessController.hasAccess({ address: deployerAddr })).toBe(false)
    },
    TIMEOUT,
  )

  it(
    'Add/Remove Access',
    async () => {
      const addCmd = new AddAccess(
        {
          ...CMD_FLAGS,
          address: aliceAddr,
        },
        [ACAddr],
      )
      await addCmd.run()

      expect(await AccessController.hasAccess({ address: aliceAddr })).toBe(true)

      const removeCmd = new RemoveAccess(
        {
          ...CMD_FLAGS,
          address: aliceAddr,
        },
        [ACAddr],
      )
      await removeCmd.run()

      expect(await AccessController.hasAccess({ address: aliceAddr })).toBe(false)
    },
    TIMEOUT,
  )
})
