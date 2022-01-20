import { inspection } from '@chainlink/gauntlet-core/dist/utils'
import { getRDD } from '../../../../lib/rdd'
import { InspectInstruction, InspectionInput, instructionToInspectCommand } from '../../../abstract/inspectionWrapper'
import { getOffchainConfigInput, OffchainConfig } from '../setConfig'

type Expected = {
  description: string
  decimals: string | number
  minAnswer: string | number
  maxAnswer: string | number
  transmitters: string[]
  billingAccessController: string
  requesterAccessController: string
  link: string
  billing: {
    observationPaymentGjuels: string
    recommendedGasPrice: string
    transmissionPaymentGjuels: string
  }
  offchainConfig: OffchainConfig
}

const makeInput = async (flags: any, args: string[]): Promise<InspectionInput<any, Expected>> => {
  if (flags.input) return flags.input as InspectionInput<any, Expected>
  const rdd = getRDD(flags.rdd)
  const contract = args[0]
  const info = rdd.contracts[contract]
  const aggregatorOperators: string[] = info.oracles.map((o) => o.operator)
  const transmitters = aggregatorOperators.map((operator) => rdd.operators[operator].ocrNodeAddress[0])
  const billingAccessController = flags.billingAccessController || process.env.BILLING_ACCESS_CONTROLLER
  const requesterAccessController = flags.requesterAccessController || process.env.REQUESTER_ACCESS_CONTROLLER
  const link = flags.link || process.env.LINK
  const offchainConfig = getOffchainConfigInput(rdd, contract)
  return {
    expected: {
      description: info.name,
      decimals: info.decimals,
      minAnswer: info.minSubmissionValue,
      maxAnswer: info.maxSubmissionValue,
      transmitters,
      billingAccessController,
      requesterAccessController,
      link,
      offchainConfig,
      billing: {
        observationPaymentGjuels: info.billing.observationPaymentGjuels,
        recommendedGasPrice: info.billing.recommendedGasPrice,
        transmissionPaymentGjuels: info.billing.transmissionPaymentGjuels,
      },
    },
  }
}

const makeOnchainData = (instructionsData: any[]): Expected => {
  const latestConfigDetails = instructionsData[0]
  // TODO: Offchain config is not stored onchain, only the digested config. Gauntlet could calculate the digested with RDD values and compare it
  // const offchainConfig = deserializeConfig(latestConfigDetails.config_digest)
  const description = instructionsData[1]
  const transmitters = instructionsData[2]
  const decimals = instructionsData[3]
  const billing = instructionsData[4]
  const billingAC = instructionsData[5]
  const requesterAC = instructionsData[6]
  const link = instructionsData[7]

  return {
    description,
    decimals,
    minAnswer: 'INFO NOT AVAILABLE IN CONTRACT',
    maxAnswer: 'INFO NOT AVAILABLE IN CONTRACT',
    transmitters: transmitters.addresses,
    billingAccessController: billingAC,
    requesterAccessController: requesterAC,
    link,
    offchainConfig: {} as OffchainConfig,
    billing: {
      observationPaymentGjuels: billing.observation_payment,
      transmissionPaymentGjuels: 'INFO NOT AVAILABLE IN CONTRACT',
      recommendedGasPrice: billing.recommended_gas_price,
    },
  }
}

const inspect = (expected: Expected, onchainData: Expected): boolean => {
  const inspections: inspection.Inspection[] = [
    inspection.makeInspection(onchainData.description, expected.description, 'Description'),
    inspection.makeInspection(onchainData.decimals, expected.decimals, 'Decimals'),
    inspection.makeInspection(onchainData.transmitters, expected.transmitters, 'Transmitters'),
    inspection.makeInspection(
      onchainData.billingAccessController,
      expected.billingAccessController,
      'Billing Access Controller',
    ),
    inspection.makeInspection(
      onchainData.requesterAccessController,
      expected.requesterAccessController,
      'Requester Access Controller',
    ),
    inspection.makeInspection(onchainData.link, expected.link, 'LINK'),
    inspection.makeInspection(onchainData.minAnswer, expected.minAnswer, 'Min Answer'),
    inspection.makeInspection(onchainData.maxAnswer, expected.maxAnswer, 'Max Answer'),
    inspection.makeInspection(
      onchainData.billing.observationPaymentGjuels,
      expected.billing.observationPaymentGjuels,
      'Observation Payment',
    ),
    inspection.makeInspection(
      onchainData.billing.recommendedGasPrice,
      expected.billing.recommendedGasPrice,
      'Recommended Gas Price',
    ),
    inspection.makeInspection(
      onchainData.billing.transmissionPaymentGjuels,
      expected.billing.transmissionPaymentGjuels,
      'Transmission Payment',
    ),
  ]
  return inspection.inspect(inspections)
}

const instruction: InspectInstruction<any, Expected> = {
  command: {
    contract: 'ocr2',
    id: 'inspect',
  },
  instructions: [
    {
      contract: 'ocr2',
      function: 'latest_config_details',
    },
    {
      contract: 'ocr2',
      function: 'description',
    },
    {
      contract: 'ocr2',
      function: 'transmitters',
    },
    {
      contract: 'ocr2',
      function: 'decimals',
    },
    {
      contract: 'ocr2',
      function: 'billing',
    },
    {
      contract: 'ocr2',
      function: 'billing_access_controller',
    },
    {
      contract: 'ocr2',
      function: 'requester_access_controller',
    },
    {
      contract: 'ocr2',
      function: 'link_token',
    },
  ],
  makeInput,
  makeOnchainData,
  inspect,
}

export default instructionToInspectCommand<any, Expected>(instruction)
