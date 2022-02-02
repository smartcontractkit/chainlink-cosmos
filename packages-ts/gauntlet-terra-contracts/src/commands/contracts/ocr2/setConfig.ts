import { getRDD } from '../../../lib/rdd'
import { AbstractInstruction, instructionToCommand } from '../../abstract/wrapper'
import { time, BN } from '@chainlink/gauntlet-core/dist/utils'
import { serializeOffchainConfig } from '../../../lib/encoding'
import { ORACLES_MAX_LENGTH } from '../../../lib/constants'

type ContractInput = {
  signers: string[]
  transmitters: string[]
  f: number
  onchain_config: string
  offchain_config_version: number
  offchain_config: string
}

type CommandInput = {
  signers: string[]
  transmitters: string[]
  offchainConfig: OffchainConfig
  offchainConfigVersion: number
  onchainConfig: OnchainConfig
}

type OnchainConfig = any

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

const getOffchainConfigInput = (rdd: any, contract: string): OffchainConfig => {
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

// Command Input is what the user needs to provide to the command to work
const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  const rdd = getRDD(flags.rdd)
  const contract = args[0]
  const aggregator = rdd.contracts[contract]
  const aggregatorOperators: any[] = aggregator.oracles.map((o) => rdd.operators[o.operator])
  const signers = aggregatorOperators.map((o) => o.ocr2OnchainPublicKey[0].replace('ocr2on_terra_', ''))
  const transmitters = aggregatorOperators.map((o) => o.ocrNodeAddress[0])

  return {
    signers,
    transmitters,
    offchainConfig: getOffchainConfigInput(rdd, contract),
    offchainConfigVersion: 2,
    onchainConfig: '',
  }
}

// Transforms the user input to a valid input for the contract function
const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  const offchainConfig = await serializeOffchainConfig(input.offchainConfig)
  const signers = input.signers.map((s) => Buffer.from(s, 'hex').toString('base64'))

  return {
    signers: signers,
    transmitters: input.transmitters,
    f: input.offchainConfig.f,
    onchain_config: input.onchainConfig,
    offchain_config_version: 2,
    offchain_config: offchainConfig,
  }
}

const validateInput = (input: CommandInput): boolean => {
  const { offchainConfig } = input
  if (3 * offchainConfig.f >= input.signers.length)
    throw new Error(
      `Signers length needs to be higher than 3 * f (${3 * offchainConfig.f}). Currently ${input.signers.length}`,
    )

  if (input.signers.length !== input.transmitters.length)
    throw new Error(`Signers and Trasmitters length are different`)

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
  instruction: {
    contract: 'ocr2',
    function: 'set_config',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(instruction)
