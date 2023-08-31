import { CosmWasmClient, SigningCosmWasmClient } from '@cosmjs/cosmwasm-stargate'
import Inspect from '../../src/commands/contracts/ocr2/inspection/inspect'
import { CMD_FLAGS, DEFAULT_GAS_PRICE, NODE_URL, TIMEOUT, deployAC, deployLink, deployOCR2, initWasmd } from '../utils'
import { GasPrice } from '@cosmjs/stargate'
import Operators from '../../src/commands/contracts/ocr2/inspection/operators'
import Responses from '../../src/commands/contracts/ocr2/inspection/responses'

describe('OCR Inspection', () => {
  let ocr2Addr: string
  let linkAddr: string
  let billingACAddr: string
  let requesterACAddr: string

  beforeAll(async () => {
    // Ideally, we'd start wasmd beforeEach() but it takes too long
    await initWasmd()

    // deploy access controller and link tokens
    linkAddr = await deployLink()
    billingACAddr = await deployAC()
    requesterACAddr = await deployAC()

    // just give two non-contract addresses
    ocr2Addr = await deployOCR2({
      minSubmissionValue: '1',
      maxSubmissionValue: '3',
      decimals: '18',
      name: 'yessir it is ocr2!',
      billingAccessController: billingACAddr,
      requesterAccessController: requesterACAddr,
      link: linkAddr,
    })
  }, TIMEOUT)

  it('Inspects OCR2 Values', async () => {
    await new Inspect(
      {
        ...CMD_FLAGS,
        name: 'yessir it is ocr2!',
        decimals: '18',
        transmitters: [],
        billingAccessController: billingACAddr,
        requesterAccessController: requesterACAddr,
        link: linkAddr,
        observationPaymentGjuels: '0',
        recommendedGasPriceMicro: '0',
        transmissionPaymentGjuels: '0',
      },
      [ocr2Addr],
    ).run()
  })

  it('Transmission Info', async () => {
    await new Operators(
      {
        ...CMD_FLAGS,
      },
      [ocr2Addr],
    ).run()
  })

  it(
    'Inspect Round Data',
    async () => {
      try {
        // will fail because transmit() has not been called (cosmos does not zero-initialize storage)
        await new Responses(
          {
            ...CMD_FLAGS,
            transmitters: [],
            operators: [],
            apis: [],
            description: 'yessir it is ocr2!',
            aggregatorOracles: [],
          },
          [ocr2Addr],
        ).run()
      } catch (e) {
        expect((e.message as string).match(/[\S\s]*ocr2::state::Transmission not found[\S\s]*/)).not.toBeNull()
      }
    },
    TIMEOUT,
  )
})
