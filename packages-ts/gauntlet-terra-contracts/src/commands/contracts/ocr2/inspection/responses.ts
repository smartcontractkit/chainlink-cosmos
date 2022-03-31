import { RDD } from '@chainlink/gauntlet-terra'
import { InspectInstruction, instructionToInspectCommand } from '../../../abstract/inspectionWrapper'
import { CONTRACT_LIST } from '../../../../lib/contracts'
import { CATEGORIES } from '../../../../lib/constants'
import { LCDClient } from '@terra-money/terra.js'
import { getLatestOCRNewTransmissionEvent, parseObserversByLength, RoundData } from '../../../../lib/inspection'
import { inspection, logger } from '@chainlink/gauntlet-core/dist/utils'

type CommandInput = {
  transmitters: string[]
  description: string
  observersInfo: { [address: string]: ObserverInfo }
}

type ContractExpectedInfo = {
  transmitters: string[]
  description: string
  aggregatorAddress: string
  latestRoundData: RoundData
  event: NewTransmissionEventData
}

type ObserverInfo = {
  name: string
  apis: string
  observation?: number
}

type NewTransmissionEventData = {
  answer: number
  transmitter: string
  observations: number[]
  observers: string
  observations_timestamp: number
  aggregator_round_id: number
}

const makeInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  const rdd = RDD.getRDD(flags.rdd)
  const operators = rdd.operators
  const aggregatorProps = rdd.contracts[args[0]]
  const aggregatorOracles = aggregatorProps['oracles']
  const aggregatorOperators: string[] = aggregatorOracles.map((o) => o.operator)
  const transmitters = aggregatorOperators.map((o) => rdd.operators[o].ocrNodeAddress[0])

  const observersInfo: { [address: string]: ObserverInfo } = {}
  aggregatorOracles.forEach((oracle) => {
    observersInfo[operators[oracle.operator].ocrNodeAddress[0]] = {
      name: oracle.operator,
      apis: oracle.api,
    }
  })

  return {
    transmitters,
    description: aggregatorProps.name,
    observersInfo,
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

  let event
  const onchainEvent = await getLatestOCRNewTransmissionEvent(provider, aggregator)
  if (onchainEvent) {
    event = {
      answer: Number(onchainEvent.answer[0]),
      transmitter: onchainEvent.transmitter[0],
      observations: onchainEvent.observations,
      observers: onchainEvent.observers[0],
      observations_timestamp: Number(onchainEvent.observations_timestamp[0]),
      aggregator_round_id: Number(onchainEvent.aggregator_round_id[0]),
    }
  }

  return {
    transmitters: transmitters.addresses,
    description,
    aggregatorAddress: aggregator,
    latestRoundData: {
      roundId: latestRoundData.round_id,
      answer: latestRoundData.answer,
      observationsTimestamp: latestRoundData.observations_timestamp,
      transmissionTimestamp: latestRoundData.transmission_timestamp,
    },
    event: event,
  }
}

const inspect = (inputData: CommandInput, onchainData: ContractExpectedInfo): boolean => {
  let inspections: inspection.Inspection[] = [
    inspection.makeInspection(onchainData.transmitters, inputData.transmitters, 'Transmitters'),
    inspection.makeInspection(onchainData.description, inputData.description, 'Description'),
  ]
  inspection.inspect(inspections)

  const event = onchainData.event
  if (!event) {
    logger.error(
      `Could not find NewTransmission event of aggregator ${onchainData.aggregatorAddress} for the latest round).`,
    )
    return false
  }
  logger.debug(event)
  logger.debug(onchainData.transmitters)

  const roundTime = new Date(Number(event.observations_timestamp) * 1000)
  logger.info(`Found NewTransmission event for round ${event.aggregator_round_id} (${roundTime})`)

  let responsiveNodes: { [address: string]: ObserverInfo } = {}
  let unresponsiveNodes: { [address: string]: ObserverInfo } = {}

  const parsedObserversIndexes = parseObserversByLength(event.observers, event.observations.length)

  parsedObserversIndexes.forEach((val, i) => {
    const observation = event.observations[i]
    const nodeAddress = onchainData.transmitters[val]
    if (nodeAddress in inputData.observersInfo) {
      responsiveNodes[nodeAddress] = {
        name: inputData.observersInfo[nodeAddress]['name'],
        apis: inputData.observersInfo[nodeAddress]['apis'],
        observation,
      }
    } else {
      logger.error(`The oracle address was not found in the input data: ${nodeAddress}`)
    }
  })

  logger.success(`${Object.keys(responsiveNodes).length} out of ${onchainData.transmitters.length} nodes responded:`)
  logger.log(responsiveNodes)

  // Count missing responses
  onchainData.transmitters.forEach((address) => {
    if (!(address in responsiveNodes)) {
      if (address in inputData.observersInfo) {
        unresponsiveNodes[address] = {
          name: inputData.observersInfo[address]['name'],
          apis: inputData.observersInfo[address]['apis'],
        }
      } else {
        logger.error(`The oracle address was not found in the input data: ${address}`)
      }
    }
  })

  if (Object.keys(unresponsiveNodes).length > 0) {
    logger.warn(`Found ${Object.keys(unresponsiveNodes).length} unresponsive nodes:`)
    logger.log(unresponsiveNodes)
  } else {
    logger.success(`All ${onchainData.transmitters.length} nodes responded.`)
  }

  logger.info(`Aggregated answer in round ${event.aggregator_round_id}: ${event.answer}`)

  return false
}

const instruction: InspectInstruction<CommandInput, ContractExpectedInfo> = {
  command: {
    category: CATEGORIES.OCR,
    contract: CONTRACT_LIST.OCR_2,
    id: 'inspect:responses',
    examples: [
      'ocr2:inspect:responses --rdd=[PATH TO RDD] <AGGREGATOR_ADDRESS>',
      'ocr2:inspect:responses --rdd=[PATH TO RDD] --round=[ROUND NUMBER] <AGGREGATOR_ADDRESS>',
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
  makeOnchainData,
  inspect,
}

export default instructionToInspectCommand<CommandInput, ContractExpectedInfo>(instruction)
