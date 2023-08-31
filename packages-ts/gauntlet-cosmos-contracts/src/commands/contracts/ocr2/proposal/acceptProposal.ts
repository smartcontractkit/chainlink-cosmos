import { Result } from '@chainlink/gauntlet-core'
import { logger, diff } from '@chainlink/gauntlet-core/dist/utils'
import { TransactionResponse, RDD } from '@chainlink/gauntlet-cosmos'
import { encoding } from '@chainlink/gauntlet-contracts-ocr2'
import { CATEGORIES } from '../../../../lib/constants'
import {
  AbstractInstruction,
  instructionToCommand,
  BeforeExecute,
  ValidateFn,
} from '../../../abstract/executionWrapper'
import { serializeOffchainConfig, deserializeConfig } from '../../../../lib/encoding'
import { getSetConfigInputFromRDD, prepareOffchainConfigForDiff } from '../proposeOffchainConfig'
import { getLatestOCRConfigEvent } from '../../../../lib/inspection'
import assert from 'assert'

export type CommandInput = {
  proposalId: string
  digest: string
  offchainConfig: encoding.OffchainConfig
  secret: string
  randomSecret: string
}

type ContractInput = {
  id: string
  digest: string
}

const validateProposalId: ValidateFn<CommandInput> = async (input) => {
  if (!input.proposalId) throw new Error('Config Proposal ID is required')
  return true
}

const validateDigest: ValidateFn<CommandInput> = async (input) => {
  if (!input.digest) throw new Error('Config digest is required')
  return true
}

const validateRandomSecret: ValidateFn<CommandInput> = async (input) => {
  if (!input.randomSecret) throw new Error('Secret generated at proposing offchain config is required')
  return true
}

const validateOffchainConfig: ValidateFn<CommandInput> = async (input, context) => {
  const { offchainConfig } = await serializeOffchainConfig(input.offchainConfig, input.secret, input.randomSecret)
  const proposal: any = await context.provider.queryContractSmart(context.contract, {
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
  const { rdd: rddPath, secret, randomSecret } = flags

  if (!secret) {
    throw new Error('--secret flag is required.')
  }

  if (!randomSecret) {
    throw new Error('--randomSecret flag is required')
  }

  let offchainConfig = flags.offchainConfig

  if (rddPath) {
    const rdd = RDD.getRDD(rddPath)
    const contract = args[0]
    const { offchainConfig: offchainConfigRDD } = getSetConfigInputFromRDD(rdd, contract)
    offchainConfig = offchainConfigRDD
  }

  return {
    proposalId: flags.proposalId || flags.configProposal || flags.id, // --configProposal alias requested by eng ops
    digest: flags.digest,
    offchainConfig,
    secret,
    randomSecret,
  }
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context, input) => async () => {
  logger.loading(`Executing ${context.id} from contract ${context.contract}`)
  const { proposalId } = input.user

  const proposal: any = await context.provider.queryContractSmart(context.contract, {
    proposal: {
      id: proposalId,
    },
  })

  const tryDeserialize = (config: string): encoding.OffchainConfig => {
    try {
      return deserializeConfig(Buffer.from(config, 'base64'))
    } catch (e) {
      return {} as encoding.OffchainConfig
    }
  }
  // Config in Proposal
  const offchainConfigInProposal = tryDeserialize(proposal.offchain_config)
  const configInProposal = prepareOffchainConfigForDiff(offchainConfigInProposal, { f: proposal.f })

  // Config in contract
  const event = await getLatestOCRConfigEvent(context.provider, context.contract)
  const attr = event?.attributes.find(({ key }) => key === 'offchain_config')?.value
  const offchainConfigInContract = attr ? tryDeserialize(attr) : ({} as encoding.OffchainConfig)
  const configInContract = prepareOffchainConfigForDiff(offchainConfigInContract, {
    f: event?.attributes.find(({ key }) => key === 'f')?.value,
  })

  logger.info('Review the configuration difference from contract and proposal: green - added, red - deleted.')
  diff.printDiff(configInContract, configInProposal)
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
  const digest = events[0]['wasm-set_config']?.[0].attributes.find(({ key }) => key === 'latest_config_digest')?.value
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
  validations: {
    validProposalId: validateProposalId,
    validDigest: validateDigest,
    validOffchainConfig: validateOffchainConfig,
    validRandomSecret: validateRandomSecret,
  },
}

export default instructionToCommand(instruction)
