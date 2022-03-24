import { inspection, logger } from '@chainlink/gauntlet-core/dist/utils'
import { InspectInstruction, instructionToInspectCommand } from '../../../abstract/inspectionWrapper'
import { RDD } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../../../lib/constants'
import { CONTRACT_LIST } from '../../../../lib/contracts'
import { LCDClient } from '@terra-money/terra.js'

type CommandInput = {
  aggregator: string
  description: string
}

type ContractExpectedInfo = {
  aggregator: string
  description: string
  version?: string
  phaseId?: number
  decimals?: string | number
  owner?: string
  latestRoundData?: object
}

const makeInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  const rdd = RDD.getRDD(flags.rdd)
  const contract = args[0]
  const info = rdd.proxies[contract]

  return {
    aggregator: info.aggregator,
    description: info.name,
  }
}

const makeInspectionData = () => async (input: CommandInput): Promise<ContractExpectedInfo> => {
  return {
    aggregator: input.aggregator,
    description: input.description,
  }
}

const makeOnchainData = (provider: LCDClient) => async (
  instructionsData: any[],
  input: CommandInput,
  proxyContract: string,
): Promise<ContractExpectedInfo> => {
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
    latestRoundData,
  }
}

const inspect = (expected: ContractExpectedInfo, onchainData: ContractExpectedInfo): boolean => {
  let inspections: inspection.Inspection[] = [
    inspection.makeInspection(onchainData.aggregator, expected.aggregator, 'Aggregator'),
    inspection.makeInspection(onchainData.description, expected.description, 'Description'),
  ]

  logger.line()
  logger.info('Inspection results:')
  logger.info(`Aggregator: 
    - Aggregator: ${onchainData.aggregator}
  `)

  logger.info(`Version: 
    - Version: ${onchainData.version}
  `)

  logger.info(`Description: 
    - Description: ${onchainData.description}
  `)

  logger.info(`PhaseID: 
    - PhaseID: ${onchainData.phaseId}
  `)

  logger.info(`Decimals: 
    - Decimals: ${onchainData.decimals}
  `)

  logger.info(`Owner: 
    - Owner: ${onchainData.owner}
  `)

  logger.info(`Last Round Data:
    - Data: ${JSON.stringify(onchainData.latestRoundData)}
  `)

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
  makeInspectionData,
  makeOnchainData,
  inspect,
}

export default instructionToInspectCommand<CommandInput, ContractExpectedInfo>(instruction)
