import { inspection } from '@chainlink/gauntlet-core/dist/utils'
import { CONTRACT_LIST } from '../../../../lib/contracts'
import { CATEGORIES } from '../../../../lib/constants'
import { getRDD } from '../../../../lib/rdd'
import { InspectInstruction, InspectionInput, instructionToInspectCommand } from '../../../abstract/inspectionWrapper'
import { getOffchainConfigInput, OffchainConfig } from '../proposeOffchainConfig'

const MIN_LINK_AVAILABLE = '100'

type Expected = {
  description: string
  decimals: string | number
  minAnswer: string | number
  maxAnswer: string | number
  transmitters: string[]
  billingAccessController: string
  requesterAccessController: string
  link: string
  linkAvailable: string
  billing: {
    observationPaymentGjuels: string
    recommendedGasPriceMicro: string
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
      linkAvailable: MIN_LINK_AVAILABLE,
      offchainConfig,
      billing: {
        observationPaymentGjuels: info.billing.observationPaymentGjuels,
        recommendedGasPriceMicro: info.billing.recommendedGasPriceMicro,
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
  const linkAvailable = instructionsData[8]

  return {
    description,
    decimals,
    minAnswer: 'INFO NOT AVAILABLE IN CONTRACT',
    maxAnswer: 'INFO NOT AVAILABLE IN CONTRACT',
    transmitters: transmitters.addresses,
    billingAccessController: billingAC,
    requesterAccessController: requesterAC,
    link,
    linkAvailable: linkAvailable.amount,
    offchainConfig: {} as OffchainConfig,
    billing: {
      observationPaymentGjuels: billing.observation_payment_gjuels,
      transmissionPaymentGjuels: billing.transmission_payment_gjuels,
      recommendedGasPriceMicro: billing.recommended_gas_price_micro,
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
    inspection.makeInspection(onchainData.linkAvailable, expected.linkAvailable, 'LINK Available'),
    inspection.makeInspection(onchainData.minAnswer, expected.minAnswer, 'Min Answer'),
    inspection.makeInspection(onchainData.maxAnswer, expected.maxAnswer, 'Max Answer'),
    inspection.makeInspection(
      onchainData.billing.observationPaymentGjuels,
      expected.billing.observationPaymentGjuels,
      'Observation Payment',
    ),
    inspection.makeInspection(
      onchainData.billing.recommendedGasPriceMicro,
      expected.billing.recommendedGasPriceMicro,
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
    category: CATEGORIES.OCR,
    contract: CONTRACT_LIST.OCR_2,
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
    {
      contract: 'ocr2',
      function: 'link_available_for_payment',
    },
  ],
  makeInput,
  makeOnchainData,
  inspect,
}

export default instructionToInspectCommand<any, Expected>(instruction)
