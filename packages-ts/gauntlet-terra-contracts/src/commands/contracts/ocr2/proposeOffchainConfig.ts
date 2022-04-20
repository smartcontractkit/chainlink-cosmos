import { RDD } from '@chainlink/gauntlet-terra'
import { AbstractInstruction, instructionToCommand, BeforeExecute, AfterExecute } from '../../abstract/executionWrapper'
import { time, BN } from '@chainlink/gauntlet-core/dist/utils'
import { ORACLES_MAX_LENGTH } from '../../../lib/constants'
import { CATEGORIES } from '../../../lib/constants'
import { getLatestOCRConfigEvent } from '../../../lib/inspection'
import { serializeOffchainConfig, deserializeConfig, generateSecretWords } from '../../../lib/encoding'
import { logger, prompt, diff, longs } from '@chainlink/gauntlet-core/dist/utils'

type CommandInput = {
  proposalId: string
  offchainConfig: OffchainConfig
  offchainConfigVersion: number
  randomSecret?: string
}

type ContractInput = {
  id: string
  offchain_config_version: number
  offchain_config: string
}

export type OffchainConfig = {
  deltaProgressNanoseconds: number
  deltaResendNanoseconds: number
  deltaRoundNanoseconds: number
  deltaGraceNanoseconds: number
  deltaStageNanoseconds: number
  rMax: number
  f: number
  s: number[]
  offchainPublicKeys: string[]
  peerIds: string[]
  reportingPluginConfig: {
    alphaReportInfinite: boolean
    alphaReportPpb: number
    alphaAcceptInfinite: boolean
    alphaAcceptPpb: number
    deltaCNanoseconds: number
  }
  maxDurationQueryNanoseconds: number
  maxDurationObservationNanoseconds: number
  maxDurationReportNanoseconds: number
  maxDurationShouldAcceptFinalizedReportNanoseconds: number
  maxDurationShouldTransmitAcceptedReportNanoseconds: number
  configPublicKeys: string[]
}

export const getOffchainConfigInput = (rdd: any, contract: string): OffchainConfig => {
  const aggregator = rdd.contracts[contract]
  const config = aggregator.config

  const aggregatorOperators: any[] = aggregator.oracles.map((o) => rdd.operators[o.operator])
  const operatorsPublicKeys = aggregatorOperators.map((o) => o.ocr2OffchainPublicKey[0].replace('ocr2off_terra_', ''))
  const operatorsPeerIds = aggregatorOperators.map((o) => o.peerId[0])
  const operatorConfigPublicKeys = aggregatorOperators.map((o) =>
    o.ocr2ConfigPublicKey[0].replace('ocr2cfg_terra_', ''),
  )

  const input: OffchainConfig = {
    deltaProgressNanoseconds: time.durationToNanoseconds(config.deltaProgress).toNumber(),
    deltaResendNanoseconds: time.durationToNanoseconds(config.deltaResend).toNumber(),
    deltaRoundNanoseconds: time.durationToNanoseconds(config.deltaRound).toNumber(),
    deltaGraceNanoseconds: time.durationToNanoseconds(config.deltaGrace).toNumber(),
    deltaStageNanoseconds: time.durationToNanoseconds(config.deltaStage).toNumber(),
    rMax: config.rMax,
    s: config.s,
    f: config.f,
    offchainPublicKeys: operatorsPublicKeys,
    peerIds: operatorsPeerIds,
    reportingPluginConfig: {
      alphaReportInfinite: config.reportingPluginConfig.alphaReportInfinite,
      alphaReportPpb: Number(config.reportingPluginConfig.alphaReportPpb),
      alphaAcceptInfinite: config.reportingPluginConfig.alphaAcceptInfinite,
      alphaAcceptPpb: Number(config.reportingPluginConfig.alphaAcceptPpb),
      deltaCNanoseconds: time.durationToNanoseconds(config.reportingPluginConfig.deltaC).toNumber(),
    },
    maxDurationQueryNanoseconds: time.durationToNanoseconds(config.maxDurationQuery).toNumber(),
    maxDurationObservationNanoseconds: time.durationToNanoseconds(config.maxDurationObservation).toNumber(),
    maxDurationReportNanoseconds: time.durationToNanoseconds(config.maxDurationReport).toNumber(),
    maxDurationShouldAcceptFinalizedReportNanoseconds: time
      .durationToNanoseconds(config.maxDurationShouldAcceptFinalizedReport)
      .toNumber(),
    maxDurationShouldTransmitAcceptedReportNanoseconds: time
      .durationToNanoseconds(config.maxDurationShouldTransmitAcceptedReport)
      .toNumber(),
    configPublicKeys: operatorConfigPublicKeys,
  }
  return input
}

export const prepareOffchainConfigForDiff = (config: OffchainConfig, extra?: Object): Object => {
  return longs.longsInObjToNumbers({
    ...config,
    ...(extra || {}),
    offchainPublicKeys: config.offchainPublicKeys?.map((key) => Buffer.from(key).toString('hex')),
  }) as Object
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput

  if (!process.env.SECRET) {
    throw new Error('SECRET is not set in env!')
  }

  const { rdd: rddPath, randomSecret } = flags

  const rdd = RDD.getRDD(rddPath)
  const contract = args[0]

  return {
    proposalId: flags.proposalId || flags.configProposal, // -configProposal alias requested by eng ops
    offchainConfig: getOffchainConfigInput(rdd, contract),
    offchainConfigVersion: 2,
    randomSecret: randomSecret || (await generateSecretWords()),
  }
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context) => async () => {
  // Config in contract
  const event = await getLatestOCRConfigEvent(context.provider, context.contract)
  const offchainConfigInContract = event?.offchain_config
    ? deserializeConfig(Buffer.from(event.offchain_config[0], 'base64'))
    : ({} as OffchainConfig)
  const configInContract = prepareOffchainConfigForDiff(offchainConfigInContract, { f: event?.f })

  // Proposed config
  const proposedOffchainConfig = deserializeConfig(Buffer.from(context.contractInput.offchain_config, 'base64'))
  const proposedConfig = prepareOffchainConfigForDiff(proposedOffchainConfig)

  logger.info('Review the proposed changes below: green - added, red - deleted.')
  diff.printDiff(configInContract, proposedConfig)

  logger.info(
    `Important: The following secret was used to encode offchain config. You will need to provide it to approve the config proposal: 
    SECRET: ${context.input.randomSecret}`,
  )

  await prompt('Continue?')
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  const { offchainConfig } = await serializeOffchainConfig(
    input.offchainConfig,
    process.env.SECRET!,
    input.randomSecret,
  )

  return {
    id: input.proposalId,
    offchain_config_version: 2,
    offchain_config: offchainConfig.toString('base64'),
  }
}

const afterExecute: AfterExecute<CommandInput, ContractInput> = (context) => async (result): Promise<any> => {
  logger.success(`Tx succeded at ${result.responses[0].tx.hash}`)
  logger.info(
    `Important: The following secret was used to encode offchain config. You will need to provide it to approve the config proposal: 
    SECRET: ${context.input.randomSecret}`,
  )
  return {
    secret: context.input.randomSecret,
  }
}

const validateInput = (input: CommandInput): boolean => {
  const { offchainConfig } = input

  const _isNegative = (v: number): boolean => new BN(v).lt(new BN(0))
  const nonNegativeValues = [
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
  ]
  for (let prop in nonNegativeValues) {
    if (_isNegative(input[prop])) throw new Error(`${prop} must be non-negative`)
  }
  const safeIntervalNanoseconds = new BN(200).mul(time.Millisecond).toNumber()
  if (offchainConfig.deltaProgressNanoseconds < safeIntervalNanoseconds)
    throw new Error(
      `deltaProgressNanoseconds (${offchainConfig.deltaProgressNanoseconds} ns)  is set below the resource exhaustion safe interval (${safeIntervalNanoseconds} ns)`,
    )
  if (offchainConfig.deltaResendNanoseconds < safeIntervalNanoseconds)
    throw new Error(
      `deltaResendNanoseconds (${offchainConfig.deltaResendNanoseconds} ns) is set below the resource exhaustion safe interval (${safeIntervalNanoseconds} ns)`,
    )

  if (offchainConfig.deltaRoundNanoseconds >= offchainConfig.deltaProgressNanoseconds)
    throw new Error(
      `deltaRoundNanoseconds (${offchainConfig.deltaRoundNanoseconds}) must be less than deltaProgressNanoseconds (${offchainConfig.deltaProgressNanoseconds})`,
    )
  const sumMaxDurationsReportGeneration = new BN(offchainConfig.maxDurationQueryNanoseconds)
    .add(new BN(offchainConfig.maxDurationObservationNanoseconds))
    .add(new BN(offchainConfig.maxDurationReportNanoseconds))

  if (sumMaxDurationsReportGeneration.gte(new BN(offchainConfig.deltaProgressNanoseconds)))
    throw new Error(
      `sum of MaxDurationQuery/Observation/Report (${sumMaxDurationsReportGeneration}) must be less than deltaProgressNanoseconds (${offchainConfig.deltaProgressNanoseconds})`,
    )

  if (offchainConfig.rMax <= 0 || offchainConfig.rMax >= 255)
    throw new Error(`rMax (${offchainConfig.rMax}) must be greater than zero and less than 255`)

  if (offchainConfig.s.length >= 1000)
    throw new Error(`Length of S (${offchainConfig.s.length}) must be less than 1000`)
  for (let i = 0; i < offchainConfig.s.length; i++) {
    const s = offchainConfig.s[i]
    if (s < 0 || s > ORACLES_MAX_LENGTH)
      throw new Error(`S[${i}] (${s}) must be between 0 and Max Oracles (${ORACLES_MAX_LENGTH})`)
  }

  return true
}

const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  examples: [
    'yarn gauntlet ocr2:propose_offchain_config --network=NETWORK --proposalId=<PROPOSAL_ID> <CONTRACT_ADDRESS>',
  ],
  instruction: {
    category: CATEGORIES.OCR,
    contract: 'ocr2',
    function: 'propose_offchain_config',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  beforeExecute,
  afterExecute,
}

export default instructionToCommand(instruction)
