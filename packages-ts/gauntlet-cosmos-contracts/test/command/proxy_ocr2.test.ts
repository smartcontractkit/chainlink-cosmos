import { CosmWasmClient } from '@cosmjs/cosmwasm-stargate'
import { ProxyOCR2QueryClient } from '../../codegen/ProxyOCR2.client'
import { CMD_FLAGS, NODE_URL, deployAC, deployLink, deployOCR2, initWasmd, toAddr } from '../utils'
import { TIMEOUT } from '../utils'
import DeployProxyCmd from '../../src/commands/contracts/proxy_ocr2/deploy'
import ProposeCmd from '../../src/commands/contracts/proxy_ocr2/proposeContract'
import ConfirmCmd from '../../src/commands/contracts/proxy_ocr2/confirmContract'

const deployProxy = async (address: string) => {
  const cmd = new DeployProxyCmd(
    {
      ...CMD_FLAGS,
    },
    [address],
  )
  const result = await cmd.run()
  return result['responses'][0]['contract'] as string
}

describe('Proxy', () => {
  let ocr2AddrA: string
  let ocr2AddrB: string
  let proxyAddr: string
  let Proxy: ProxyOCR2QueryClient
  let deployerAddr: string

  beforeAll(async () => {
    const [deployer, mockOCRA, mockOCRB] = await initWasmd()
    deployerAddr = await toAddr(deployer)
    ocr2AddrA = await toAddr(mockOCRA)
    ocr2AddrB = await toAddr(mockOCRB)

    proxyAddr = await deployProxy(ocr2AddrA)

    const cosmClient = await CosmWasmClient.connect(NODE_URL)
    Proxy = new ProxyOCR2QueryClient(cosmClient, proxyAddr)

    expect(await Proxy.phaseId()).toEqual(1)
    expect(await Proxy.owner()).toEqual(deployerAddr)
    expect(await Proxy.aggregator()).toEqual(ocr2AddrA)
    expect(await Proxy.phaseAggregators({ phaseId: 1 })).toEqual(ocr2AddrA)
  }, TIMEOUT)

  it(
    'Propose & Confirm',
    async () => {
      await new ProposeCmd(
        {
          ...CMD_FLAGS,
          address: ocr2AddrB,
        },
        [proxyAddr],
      ).run()

      expect(await Proxy.proposedAggregator()).toEqual(ocr2AddrB)

      await new ConfirmCmd(
        {
          ...CMD_FLAGS,
          address: ocr2AddrB,
        },
        [proxyAddr],
      ).run()

      expect(await Proxy.phaseId()).toEqual(2)
      expect(await Proxy.aggregator()).toEqual(ocr2AddrB)
      expect(await Proxy.phaseAggregators({ phaseId: 2 })).toEqual(ocr2AddrB)
    },
    TIMEOUT,
  )
})
