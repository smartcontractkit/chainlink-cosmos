import { RDD } from '@chainlink/gauntlet-terra'
import { InspectInstruction, instructionToInspectCommand } from '../../../abstract/inspectionWrapper'
import { CONTRACT_LIST } from '../../../../lib/contracts'
import { CATEGORIES } from '../../../../lib/constants'
import { LCDClient } from '@terra-money/terra.js'
import { getLatestOCRNewTransmissionEvent, parseObservers, RoundData } from '../../../../lib/inspection'
import { inspection, logger } from '@chainlink/gauntlet-core/dist/utils'

type CommandInput = {
  transmitters: string[]
  description: string
  operators: string[]
  aggregatorOracles: any[]
}

type ContractExpectedInfo = {
  transmitters: string[]
  description: string
  operators: string[]
  aggregatorOracles: any[]
  aggregatorAddress?: string
  latestRoundData?: RoundData
  event?: any
}

type ObservationSummary = {
  nodeAddress: string
  name: string
  submission: string
  apis: string
}

const makeInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  const rdd = RDD.getRDD(flags.rdd)
  const aggregatorAddress = args[0]
  const aggregatorProps = rdd.contracts[aggregatorAddress]
  const aggregatorOracles = aggregatorProps['oracles']
  const aggregatorOperators: string[] = aggregatorOracles.map((o) => o.operator)

  const operators = rdd.operators
  const transmitters = aggregatorOperators.map((o) => rdd.operators[o].ocrNodeAddress[0])

  return {
    transmitters,
    description: aggregatorProps.name,
    operators,
    aggregatorOracles,
  }
}

const makeInspectionData = () => async (input: CommandInput): Promise<ContractExpectedInfo> => {
  return {
    transmitters: input.transmitters,
    description: input.description,
    operators: input.operators,
    aggregatorOracles: input.aggregatorOracles,
  }
}

const makeOnchainData = (provider: LCDClient) => async (
  instructionsData: any[],
  input: CommandInput,
  aggregator: string,
): Promise<ContractExpectedInfo> => {
  const latestRoundData = instructionsData[0]
  const transmitters = instructionsData[1]
  const description = instructionsData[2]

  const event = await getLatestOCRNewTransmissionEvent(provider, aggregator)

  return {
    transmitters: transmitters.addresses,
    description,
    operators: input.operators,
    aggregatorOracles: input.aggregatorOracles,
    aggregatorAddress: aggregator,
    latestRoundData: {
      roundId: latestRoundData.round_id,
      answer: latestRoundData.answer,
      observationsTimestamp: latestRoundData.observations_timestamp,
      transmissionTimestamp: latestRoundData.transmission_timestamp,
    },
    event,
  }
}

const inspect = (expected: ContractExpectedInfo, onchainData: ContractExpectedInfo): boolean => {
  let inspections: inspection.Inspection[] = [
    inspection.makeInspection(onchainData.transmitters, expected.transmitters, 'Transmitters'),
    inspection.makeInspection(onchainData.description, expected.description, 'Description'),
  ]
  if (!inspection.inspect(inspections)) {
    return false
  }

  const event = onchainData.event
  const transmitters = onchainData.transmitters
  const operators = expected.operators
  if (!event) {
    logger.error(
      `Could not find NewTransmission event of aggregator ${expected.aggregatorAddress} for the latest round).`,
    )
    return false
  }
  logger.debug(event)
  logger.debug(transmitters)

  const roundTime = new Date(Number(event.observations_timestamp) * 1000)
  logger.info(`Found NewTransmission event for round ${event.aggregator_round_id} (${roundTime})`)

  let observationSummaries = [] as ObservationSummary[]
  let responsiveNodes = [] as string[]

  const { answer, transmitter, observations, observers } = event
  const parsedObservers = parseObservers(String(observers), observations.length)

  const _findOperator = (address: string) =>
    Object.keys(operators)
      .filter((name) => 'ocrNodeAddress' in operators[name])
      .find((name) => operators[name].ocrNodeAddress.includes(address))

  parsedObservers.forEach((val, i) => {
    const submission = observations[i]
    const nodeAddress = transmitters[val]
    const name = _findOperator(nodeAddress)
    if (!name) {
      logger.error(`The oracle address was not found in the RDD: ${transmitter}`)
      return false
    }

    const op = expected.aggregatorOracles.find((o) => o.operator === name)
    const apis = op ? op['api'].join(' ') : null
    observationSummaries.push({
      nodeAddress,
      name,
      submission,
      apis,
    })
    responsiveNodes.push(nodeAddress)
  })

  logger.success(`${parsedObservers.length} out of ${transmitters.length} nodes responded:`)
  logger.log(observationSummaries)

  // Count missing responses
  const missingNodes = transmitters.filter((n) => !responsiveNodes.includes(n))
  const missingResponses = missingNodes.map((n) => {
    const name = _findOperator(n)
    const op = expected.aggregatorOracles.find((o) => o.operator === name)
    const apis = op ? op['api'].join(' ') : null
    return {
      nodeAddress: n,
      name,
      apis,
    }
  })
  if (missingResponses.length > 0) {
    logger.warn(`Found ${missingResponses.length} unresponsive nodes:`)
    logger.log(missingResponses)
  } else {
    logger.success(`All ${transmitters.length} nodes responded.`)
  }

  logger.info(`Aggregated answer in round ${event.aggregator_round_id}: ${answer}`)

  return false
}

const instruction: InspectInstruction<CommandInput, ContractExpectedInfo> = {
  command: {
    category: CATEGORIES.OCR,
    contract: CONTRACT_LIST.OCR_2,
    id: 'responses',
    examples: [
      'ocr2:responses --rdd=[PATH TO RDD] <AGGREGATOR_ADDRESS>',
      'ocr2:responses --rdd=[PATH TO RDD] --round=[ROUND NUMBER] <AGGREGATOR_ADDRESS>',
    ],
  },
  instructions: [
    {
      contract: 'ocr2',
      function: 'latest_round_data',
    },
    {
      contract: 'ocr2',
      function: 'transmitters',
    },
    {
      contract: 'ocr2',
      function: 'description',
    },
  ],
  makeInput,
  makeInspectionData,
  makeOnchainData,
  inspect,
}

export default instructionToInspectCommand<CommandInput, ContractExpectedInfo>(instruction)
