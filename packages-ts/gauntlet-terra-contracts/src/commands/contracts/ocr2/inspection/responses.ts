import { RDD } from '@chainlink/gauntlet-terra'
import { InspectInstruction, instructionToInspectCommand } from '../../../abstract/inspectionWrapper'
import { CONTRACT_LIST } from '../../../../lib/contracts'
import { CATEGORIES } from '../../../../lib/constants'
import { LCDClient } from '@terra-money/terra.js'
import { getLatestOCRNewTransmissionEvent, RoundData } from '../../../../lib/inspection'
import { inspection, logger } from '@chainlink/gauntlet-core/dist/utils'
import { dateFromUnix } from '../../../../lib/utils'

type CommandInput = {
  transmitters: string[]
  description: string
  aggregatorOracles: Oracle[]
}

type ContractExpectedInfo = {
  transmitters: string[]
  description: string
  aggregatorAddress: string
  latestRoundData: RoundData
  event?: NewTransmissionEventData
}

type Oracle = {
  transmitter: string
  name: string
  apis: string[]
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
  const oracles = aggregatorProps['oracles']
  const aggregatorOperators: string[] = oracles.map((o) => o.operator)
  const transmitters = aggregatorOperators.map((o) => rdd.operators[o].ocrNodeAddress[0])

  const aggregatorOracles = oracles.map((o) => {
    return {
      transmitter: operators[o.operator].ocrNodeAddress[0],
      name: o.operator,
      apis: o.api,
    }
  })

  return {
    transmitters,
    description: aggregatorProps.name,
    aggregatorOracles,
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

  const onchainEvent = await getLatestOCRNewTransmissionEvent(provider, aggregator)

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
    ...(onchainEvent && {
      event: {
        answer: Number(onchainEvent.answer[0]),
        transmitter: onchainEvent.transmitter[0],
        observations: onchainEvent.observations.map((o) => Number(o)),
        observers: onchainEvent.observers[0],
        observations_timestamp: Number(onchainEvent.observations_timestamp[0]),
        aggregator_round_id: Number(onchainEvent.aggregator_round_id[0]),
      },
    }),
  }
}

const parseObserversByLength = (observers: string, observersNumber: number): number[] =>
  (observers.substring(0, observersNumber * 2).match(/.{2}/g) || []).map((s) => parseInt(s, 16))

const inspect = (inputData: CommandInput, onchainData: ContractExpectedInfo): boolean => {
  let inspections: inspection.Inspection[] = [
    inspection.makeInspection(onchainData.transmitters, inputData.transmitters, 'Transmitters'),
    inspection.makeInspection(onchainData.description, inputData.description, 'Description'),
  ]

  const event = onchainData.event
  if (!event) {
    logger.error(
      `Could not find NewTransmission event of aggregator ${onchainData.aggregatorAddress} for the latest round).`,
    )
    return false
  }
  logger.debug(event)
  logger.debug(onchainData.transmitters)

  const roundTime = dateFromUnix(event.observations_timestamp)
  logger.info(`Found NewTransmission event for round ${event.aggregator_round_id} (${roundTime})`)

  const parsedObserversIndexes = parseObserversByLength(event.observers, event.observations.length)

  const responsiveNodes = parsedObserversIndexes
    .filter((val) =>
      inputData.aggregatorOracles.find(({ transmitter }) => transmitter == onchainData.transmitters[val]),
    )
    .map((val, i) => {
      const nodeAddress = onchainData.transmitters[val]
      const oracle = inputData.aggregatorOracles.find(({ transmitter }) => transmitter == nodeAddress)
      return {
        transmitter: nodeAddress,
        name: oracle?.name,
        apis: oracle?.apis,
        observation: event.observations[i],
      } as Oracle
    })

  logger.success(`${responsiveNodes.length} out of ${onchainData.transmitters.length} nodes responded:`)
  logger.log(responsiveNodes)

  // Count missing responses
  const unresponsiveNodes = inputData.aggregatorOracles.filter(
    (aggregatorOracle) => !responsiveNodes.some((oracle) => oracle.transmitter == aggregatorOracle.transmitter),
  )

  if (unresponsiveNodes.length > 0) {
    logger.warn(`Found ${unresponsiveNodes.length} unresponsive nodes:`)
    logger.log(unresponsiveNodes)
  } else {
    logger.success(`All ${onchainData.transmitters.length} nodes responded.`)
  }

  logger.info(`Aggregated answer in round ${event.aggregator_round_id}: ${event.answer}`)

  return inspection.inspect(inspections)
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
