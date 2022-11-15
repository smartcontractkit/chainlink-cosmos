import { BN, inspection, logger, longs } from '@chainlink/gauntlet-core/dist/utils'
import { RDD, Client } from '@chainlink/gauntlet-terra'
import { CONTRACT_LIST } from '../../../../lib/contracts'
import { CATEGORIES, TOKEN_UNIT } from '../../../../lib/constants'
import { InspectInstruction, instructionToInspectCommand } from '../../../abstract/inspectionWrapper'
import { deserializeConfig } from '../../../../lib/encoding'
import { getOffchainConfigInput, OffchainConfig } from '../proposeOffchainConfig'
import { getLatestOCRConfigEvent } from '../../../../lib/inspection'

// Command input and expected info is the same here
type ContractExpectedInfo = {
  description: string
  decimals: string | number
  transmitters: string[]
  billingAccessController: string
  requesterAccessController: string
  link: string
  billing: {
    observationPaymentGjuels: string
    recommendedGasPriceMicro: string
    transmissionPaymentGjuels: string
  }
  offchainConfig: OffchainConfig
  totalOwed?: string
  linkAvailable?: string
  owner?: string
}

const makeInput = async (flags: any, args: string[]): Promise<ContractExpectedInfo> => {
  if (flags.input) return flags.input as ContractExpectedInfo
  const rdd = RDD.getRDD(flags.rdd)
  const contract = args[0]
  const info = rdd.contracts[contract]
  const aggregatorOperators: string[] = info.oracles.map((o) => o.operator)
  const transmitters = aggregatorOperators.map((o) => rdd.operators[o].ocrNodeAddress[0])
  const billingAccessController = flags.billingAccessController || process.env.BILLING_ACCESS_CONTROLLER
  const requesterAccessController = flags.requesterAccessController || process.env.REQUESTER_ACCESS_CONTROLLER
  const link = flags.link || process.env.LINK

  return {
    description: info.name,
    decimals: info.decimals,
    transmitters,
    billingAccessController,
    requesterAccessController,
    link,
    billing: {
      observationPaymentGjuels: info.billing.observationPaymentGjuels,
      recommendedGasPriceMicro: info.billing.recommendedGasPriceMicro,
      transmissionPaymentGjuels: info.billing.transmissionPaymentGjuels,
    },
    offchainConfig: getOffchainConfigInput(rdd, contract),
  }
}

const makeOnchainData = (provider: Client) => async (
  instructionsData: any[],
  input: ContractExpectedInfo,
  aggregator: string,
): Promise<ContractExpectedInfo> => {
  const latestConfigDetails = instructionsData[0]
  const description = instructionsData[1]
  const transmitters = instructionsData[2]
  const decimals = instructionsData[3]
  const billing = instructionsData[4]
  const billingAC = instructionsData[5]
  const requesterAC = instructionsData[6]
  const link = instructionsData[7]
  const linkAvailable = instructionsData[8]
  const owner = instructionsData[9]

  const owedPerTransmitter: string[] = await Promise.all(
    transmitters.addresses.map((t) => {
      return provider.queryContractSmart(aggregator, {
        owed_payment: {
          transmitter: t,
        },
      })
    }),
  )

  const event = await getLatestOCRConfigEvent(provider, aggregator)
  const offchain_config = event?.attributes.find(({ key }) => key === 'offchain_config')?.value
  let offchainConfig = {} as OffchainConfig

  if (offchain_config) {
    try {
      offchainConfig = deserializeConfig(Buffer.from(offchain_config, 'base64'))
    } catch (e) {
      logger.warn('Could not deserialize offchain config')
    }
  }

  const totalOwed = owedPerTransmitter.reduce((agg: BN, v) => agg.add(new BN(v)), new BN(0)).toString()
  return {
    description,
    decimals,
    transmitters: transmitters.addresses,
    billingAccessController: billingAC,
    requesterAccessController: requesterAC,
    link,
    linkAvailable: linkAvailable.amount,
    billing: {
      observationPaymentGjuels: billing.observation_payment_gjuels,
      transmissionPaymentGjuels: billing.transmission_payment_gjuels,
      recommendedGasPriceMicro: billing.recommended_gas_price_micro,
    },
    totalOwed,
    owner,
    offchainConfig,
  }
}

const inspect = (expected: ContractExpectedInfo, onchainData: ContractExpectedInfo): boolean => {
  let inspections: inspection.Inspection[] = [
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

  if (!!onchainData.offchainConfig.s) {
    const offchainConfigInspections: inspection.Inspection[] = [
      inspection.makeInspection(onchainData.offchainConfig.s, expected.offchainConfig.s, 'Offchain Config "s"'),
      inspection.makeInspection(
        onchainData.offchainConfig.peerIds,
        expected.offchainConfig.peerIds,
        'Offchain Config "peerIds"',
      ),
      inspection.makeInspection(
        longs.toComparableNumber(onchainData.offchainConfig.rMax),
        longs.toComparableNumber(expected.offchainConfig.rMax),
        'Offchain Config "rMax"',
      ),
      inspection.makeInspection(
        onchainData.offchainConfig.offchainPublicKeys.map((k) => Buffer.from(k).toString('hex')),
        expected.offchainConfig.offchainPublicKeys,
        `Offchain Config "offchainPublicKeys"`,
      ),
      inspection.makeInspection(
        onchainData.offchainConfig.reportingPluginConfig.alphaReportInfinite,
        expected.offchainConfig.reportingPluginConfig.alphaReportInfinite,
        'Offchain Config "reportingPluginConfig.alphaReportInfinite"',
      ),
      inspection.makeInspection(
        onchainData.offchainConfig.reportingPluginConfig.alphaAcceptInfinite,
        expected.offchainConfig.reportingPluginConfig.alphaAcceptInfinite,
        'Offchain Config "reportingPluginConfig.alphaAcceptInfinite"',
      ),
      inspection.makeInspection(
        longs.toComparableNumber(onchainData.offchainConfig.reportingPluginConfig.alphaReportPpb),
        longs.toComparableNumber(expected.offchainConfig.reportingPluginConfig.alphaReportPpb),
        `Offchain Config "reportingPluginConfig.alphaReportPpb"`,
      ),
      inspection.makeInspection(
        longs.toComparableNumber(onchainData.offchainConfig.reportingPluginConfig.alphaAcceptPpb),
        longs.toComparableNumber(expected.offchainConfig.reportingPluginConfig.alphaAcceptPpb),
        `Offchain Config "reportingPluginConfig.alphaAcceptPpb"`,
      ),
      inspection.makeInspection(
        longs.toComparableNumber(onchainData.offchainConfig.reportingPluginConfig.deltaCNanoseconds),
        longs.toComparableNumber(expected.offchainConfig.reportingPluginConfig.deltaCNanoseconds),
        `Offchain Config "reportingPluginConfig.deltaCNanoseconds"`,
      ),
    ]

    const longNumberInspections = [
      'deltaProgressNanoseconds',
      'deltaResendNanoseconds',
      'deltaRoundNanoseconds',
      'deltaGraceNanoseconds',
      'deltaStageNanoseconds',
      'maxDurationQueryNanoseconds',
      'maxDurationObservationNanoseconds',
      'maxDurationReportNanoseconds',
      'maxDurationShouldAcceptFinalizedReportNanoseconds',
      'maxDurationShouldTransmitAcceptedReportNanoseconds',
    ].map((prop) =>
      inspection.makeInspection(
        longs.toComparableNumber(onchainData.offchainConfig[prop]),
        longs.toComparableNumber(expected.offchainConfig[prop]),
        `Offchain Config "${prop}"`,
      ),
    )

    inspections = inspections.concat(offchainConfigInspections).concat(longNumberInspections)
  } else {
    logger.error('Could not get offchain config information from the contract. Skipping offchain config inspection')
  }

  logger.line()
  logger.info('Inspection results:')
  logger.info(`Ownership: 
    - Owner: ${onchainData.owner}
  `)
  logger.info(`Funding:
    - LINK Available: ${onchainData.linkAvailable}
    - Total LINK Owed: ${onchainData.totalOwed}
  `)

  const owedDiff = new BN(onchainData.linkAvailable!).sub(new BN(onchainData.totalOwed!)).div(new BN(TOKEN_UNIT))
  if (owedDiff.lt(new BN(0))) {
    logger.warn(`Total LINK Owed is higher than balance. Amount to fund: ${owedDiff.mul(new BN(-1)).toString()}`)
  } else {
    logger.success(`LINK Balance can cover debt. LINK after payment: ${owedDiff.toString()}`)
  }

  const result = inspection.inspect(inspections)
  logger.line()

  return result
}

const instruction: InspectInstruction<any, ContractExpectedInfo> = {
  command: {
    category: CATEGORIES.OCR,
    contract: CONTRACT_LIST.OCR_2,
    id: 'inspect',
    examples: ['ocr2:inspect --network=<NETWORK> <CONTRACT_ADDRESS>'],
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
    {
      contract: 'ocr2',
      function: 'owner',
    },
  ],
  makeInput,
  makeOnchainData,
  inspect,
}

export default instructionToInspectCommand<any, ContractExpectedInfo>(instruction)
