import { Result } from '@chainlink/gauntlet-core'
import { logger, prompt, diff } from '@chainlink/gauntlet-core/dist/utils'
import { TransactionResponse, RDD } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../../../lib/constants'
import {
  AbstractInstruction,
  instructionToCommand,
  BeforeExecute,
  Validate,
  makeValidations,
} from '../../../abstract/executionWrapper'
import { serializeOffchainConfig, deserializeConfig } from '../../../../lib/encoding'
import { getOffchainConfigInput, OffchainConfig, prepareOffchainConfigForDiff } from '../proposeOffchainConfig'
import { getLatestOCRConfigEvent } from '../../../../lib/inspection'
import assert from 'assert'

export type CommandInput = {
  proposalId: string
  digest: string
  offchainConfig: OffchainConfig
  randomSecret: string
}

type ContractInput = {
  id: string
  digest: string
}

const validationA: Validate<CommandInput> = async (input) => {
  if (!input.proposalId) throw new Error('Config Proposal ID is required')
  return true
}

const validationB: Validate<CommandInput> = async (input) => {
  if (!input.digest) throw new Error('Config digest is required')
  return true
}

const validationC: Validate<CommandInput> = async (input) => {
  if (!input.randomSecret) throw new Error('Secret generated at proposing offchain config is required')
  return true
}

const validationD: Validate<CommandInput> = async (input, context) => {
  const { offchainConfig } = await serializeOffchainConfig(
    input.offchainConfig,
    process.env.SECRET!,
    input.randomSecret,
  )
  const proposal: any = await context.provider.wasm.contractQuery(context.contract, {
    proposal: {
      id: input.proposalId,
    },
  })

  try {
    assert.equal(offchainConfig.toString('base64'), proposal.offchain_config)
  } catch (err) {
    throw new Error('Offchain config generated is different from the one proposed')
  }

  return true
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  const { rdd: rddPath, secret } = flags

  if (!secret) {
    throw new Error('--secret flag is required.')
  }

  if (!process.env.SECRET) {
    throw new Error('SECRET is not set in env!')
  }

  const rdd = RDD.getRDD(rddPath)
  const contract = args[0]

  return {
    proposalId: flags.proposalId || flags.configProposal || flags.id, // --configProposal alias requested by eng ops
    digest: flags.digest,
    offchainConfig: getOffchainConfigInput(rdd, contract),
    randomSecret: secret,
  }
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context, inputContext) => async () => {
  const { proposalId } = inputContext.input

  const proposal: any = await context.provider.wasm.contractQuery(context.contract, {
    proposal: {
      id: proposalId,
    },
  })

  const tryDeserialize = (config: string): OffchainConfig => {
    try {
      return deserializeConfig(Buffer.from(config, 'base64'))
    } catch (e) {
      return {} as OffchainConfig
    }
  }
  // Config in Proposal
  const offchainConfigInProposal = tryDeserialize(proposal.offchain_config)
  const configInProposal = prepareOffchainConfigForDiff(offchainConfigInProposal, { f: proposal.f })

  // Config in contract
  const event = await getLatestOCRConfigEvent(context.provider, context.contract)
  const offchainConfigInContract = event?.offchain_config
    ? tryDeserialize(event.offchain_config[0])
    : ({} as OffchainConfig)
  const configInContract = prepareOffchainConfigForDiff(offchainConfigInContract, { f: event?.f[0] })

  logger.info('Review the configuration difference from contract and proposal: green - added, red - deleted.')
  diff.printDiff(configInContract, configInProposal)
  await prompt('Continue?')
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    id: input.proposalId,
    digest: Buffer.from(input.digest, 'hex').toString('base64'),
  }
}

// TODO: Deprecate
const validateInput = (input: CommandInput): boolean => true

const afterExecute = () => async (response: Result<TransactionResponse>) => {
  logger.success(`Config Proposal accepted on tx ${response.responses[0].tx.hash}`)
  const events = response.responses[0].tx.events
  if (!events) {
    logger.error('Could not retrieve events from tx')
    return
  }
  const digest = events[0]['wasm-set_config'].latest_config_digest[0]
  return {
    digest,
  }
}

export const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  examples: [
    'yarn gauntlet ocr2:accept_proposal --network=<NETWORK> --configProposal=<PROPOSAL_ID> --digest=<DIGEST> <CONTRACT_ADDRESS>',
  ],
  instruction: {
    contract: 'ocr2',
    function: 'accept_proposal',
    category: CATEGORIES.OCR,
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  beforeExecute,
  afterExecute,
  validations: makeValidations(validationA, validationB, validationC, validationD),
}

export default instructionToCommand(instruction)
