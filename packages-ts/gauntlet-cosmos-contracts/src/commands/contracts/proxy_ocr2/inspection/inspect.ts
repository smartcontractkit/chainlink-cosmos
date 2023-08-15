import { inspection, logger } from '@chainlink/gauntlet-core/dist/utils'
import { InspectInstruction, instructionToInspectCommand } from '../../../abstract/inspectionWrapper'
import { RDD, Client } from '@chainlink/gauntlet-cosmos'
import { CATEGORIES } from '../../../../lib/constants'
import { CONTRACT_LIST } from '../../../../lib/contracts'
import { dateFromUnix } from '../../../../lib/utils'
import { RoundData } from '../../../../lib/inspection'

type CommandInput = {
  aggregator: string
  description: string
  decimals: string | number
}

type ContractExpectedInfo = {
  aggregator: string
  description: string
  decimals: string | number
  version?: string
  phaseId?: number
  owner?: string
  latestRoundData?: RoundData
}

const makeInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput

  if (flags.rdd) {
    const rdd = RDD.getRDD(flags.rdd)
    const contract = args[0]
    const info = rdd.proxies[contract]
    const decimals = rdd.contracts[rdd.proxies[contract].aggregator].decimals

    return {
      aggregator: info.aggregator,
      description: info.name,
      decimals,
    }
  }

  return {
    aggregator: args[0],
    description: flags.description || '',
    decimals: flags.decimals || '',
  }
}

const makeOnchainData =
  (provider: Client) =>
  async (instructionsData: any[], input: CommandInput, proxyContract: string): Promise<ContractExpectedInfo> => {
    const aggregator = instructionsData[0]
    const version = instructionsData[1]
    const description = instructionsData[2]
    const phaseId = instructionsData[3]
    const decimals = instructionsData[4]
    const owner = instructionsData[5]
    const latestRoundData = instructionsData[6]

    return {
      aggregator,
      version,
      description,
      phaseId,
      decimals,
      owner,
      latestRoundData: {
        roundId: latestRoundData.round_id,
        answer: latestRoundData.answer,
        observationsTimestamp: latestRoundData.observations_timestamp,
        transmissionTimestamp: latestRoundData.transmission_timestamp,
      },
    }
  }

const inspect = (expected: CommandInput, onchainData: ContractExpectedInfo): boolean => {
  let inspections: inspection.Inspection[] = [
    inspection.makeInspection(onchainData.aggregator, expected.aggregator, 'Aggregator'),
    inspection.makeInspection(onchainData.description, expected.description, 'Description'),
    inspection.makeInspection(onchainData.decimals, expected.decimals, 'Decimals'),
  ]

  logger.line()
  logger.info('Inspection results:')
  logger.info(`Aggregator: 
    - Address: ${onchainData.aggregator}
    - Version: ${onchainData.version}
    - Description: ${onchainData.description}
    - PhaseID: ${onchainData.phaseId}
    - Decimals: ${onchainData.decimals}
    - Owner: ${onchainData.owner}
  `)

  if (onchainData.latestRoundData) {
    logger.info(`Latest Round Data:
      - Round ID: ${onchainData.latestRoundData.roundId}
      - Answer: ${Number(onchainData.latestRoundData.answer) / 10 ** Number(onchainData.decimals)}
      - Transmission at: ${dateFromUnix(onchainData.latestRoundData.transmissionTimestamp)}
      - Observation at: ${dateFromUnix(onchainData.latestRoundData.observationsTimestamp)}
    `)
  }

  const result = inspection.inspect(inspections)
  logger.line()

  return result
}

const instruction: InspectInstruction<CommandInput, ContractExpectedInfo> = {
  command: {
    category: CATEGORIES.PROXIES,
    contract: CONTRACT_LIST.PROXY_OCR_2,
    id: 'inspect',
    examples: ['proxy_ocr2:inspect --network=<NETWORK> <CONTRACT_ADDRESS>'],
  },
  instructions: [
    {
      contract: 'proxy_ocr2',
      function: 'aggregator',
    },
    {
      contract: 'proxy_ocr2',
      function: 'version',
    },
    {
      contract: 'proxy_ocr2',
      function: 'description',
    },
    {
      contract: 'proxy_ocr2',
      function: 'phase_id',
    },
    {
      contract: 'proxy_ocr2',
      function: 'decimals',
    },
    {
      contract: 'proxy_ocr2',
      function: 'owner',
    },
    {
      contract: 'proxy_ocr2',
      function: 'latest_round_data',
    },
  ],
  makeInput,
  makeOnchainData,
  inspect,
}

export default instructionToInspectCommand<CommandInput, ContractExpectedInfo>(instruction)
