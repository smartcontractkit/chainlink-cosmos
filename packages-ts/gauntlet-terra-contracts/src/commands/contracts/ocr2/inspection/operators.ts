import { InspectInstruction, instructionToInspectCommand } from '../../../abstract/inspectionWrapper'
import { CONTRACT_LIST } from '../../../../lib/contracts'
import { CATEGORIES } from '../../../../lib/constants'
import { LCDClient } from '@chainlink/gauntlet-terra'
import { getLatestOCRNewTransmissionEvents } from '../../../../lib/inspection'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { providerUtils } from '@chainlink/gauntlet-terra'
import { dateFromUnix } from '../../../../lib/utils'

type CommandInput = {}

type ContractExpectedInfo = {
  transmitters: string[]
  balances: { [key: string]: string | null }
  latestTransmissions: {
    [key: string]: {
      answer: string
      timestamp: number
    }
  }
}

const makeInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {}
}

const makeOnchainData = (provider: LCDClient) => async (
  instructionsData: any[],
  input: CommandInput,
  aggregator: string,
): Promise<ContractExpectedInfo> => {
  const transmitters = instructionsData[0]

  logger.loading('Fetching latest transmission events...')
  const events = await getLatestOCRNewTransmissionEvents(provider, aggregator, 100)

  const transmissionsPerTransmitter = events.reduce(
    (agg, e) => {
      const transmitter = e['wasm-new_transmission'].transmitter[0]
      if (agg[transmitter]) return agg
      return {
        ...agg,
        [transmitter]: {
          answer: e['wasm-new_transmission'].answer[0],
          timestamp: Number(e['wasm-new_transmission'].observations_timestamp[0]),
        },
      }
    },
    {} as {
      [key: string]: {
        answer: string
        timestamp: number
      }
    },
  )

  logger.loading('Fetching transmitters balances...')
  const balances: (string | null)[] = await Promise.all(
    transmitters.addresses.map((t) => providerUtils.getBalance(provider, t)),
  )

  return {
    transmitters: transmitters.addresses,
    balances: balances.reduce((agg, bal, i) => ({ ...agg, [transmitters.addresses[i]]: bal }), {}),
    latestTransmissions: transmissionsPerTransmitter,
  }
}

const inspect = (inputData: CommandInput, onchainData: ContractExpectedInfo): boolean => {
  const balancesMsg = Object.entries(onchainData.balances).reduce((agg, [transmitter, bal]) => {
    return `${agg}
      - Transmitter ${transmitter}: ${bal || 'Balance not found'}`
  }, 'Balances:')

  const transmissionsMsg = Object.entries(onchainData.latestTransmissions).reduce(
    (agg, [transmitter, transmission]) => {
      return `${agg}
      - Transmitter ${transmitter} transmitted at ${dateFromUnix(transmission.timestamp)} with answer ${
        transmission.answer
      }`
    },
    'Transmissions:',
  )

  logger.info(balancesMsg)
  logger.info(transmissionsMsg)

  const notResponding = onchainData.transmitters.filter(
    (t) => !Object.keys(onchainData.latestTransmissions).includes(t),
  )

  if (notResponding.length > 0) {
    logger.warn(`Transmitters that did not transmit in last 100 transmissions: 
      ${notResponding}
    `)
  }

  return true
}

const instruction: InspectInstruction<CommandInput, ContractExpectedInfo> = {
  command: {
    category: CATEGORIES.OCR,
    contract: CONTRACT_LIST.OCR_2,
    id: 'inspect:operators',
    examples: ['ocr2:inspect:operators <AGGREGATOR_ADDRESS>', 'ocr2:inspect:operators <AGGREGATOR_ADDRESS>'],
  },
  instructions: [
    {
      contract: 'ocr2',
      function: 'transmitters',
    },
  ],
  makeInput,
  makeOnchainData,
  inspect,
}

export default instructionToInspectCommand<CommandInput, ContractExpectedInfo>(instruction)
