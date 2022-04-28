import { Result } from '@chainlink/gauntlet-core'
import { logger, diff } from '@chainlink/gauntlet-core/dist/utils'
import { TransactionResponse, RDD } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../../../lib/constants'
import { AbstractInstruction, instructionToCommand, BeforeExecute } from '../../../abstract/executionWrapper'
import { serializeOffchainConfig, deserializeConfig } from '../../../../lib/encoding'
import { getOffchainConfigInput, OffchainConfig, prepareOffchainConfigForDiff } from '../proposeOffchainConfig'
import { getLatestOCRConfigEvent } from '../../../../lib/inspection'
import assert from 'assert'

type CommandInput = {
  proposalId: string
  digest: string
  offchainConfig: OffchainConfig
  randomSecret: string
}

type ContractInput = {
  id: string
  digest: string
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
    proposalId: flags.proposalId || flags.configProposal, // --configProposal alias requested by eng ops
    digest: flags.digest,
    offchainConfig: getOffchainConfigInput(rdd, contract),
    randomSecret: secret,
  }
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context) => async () => {
  logger.loading(`Executing ${context.id} from contract ${context.contract}`)

  const { proposalId, randomSecret, offchainConfig: offchainLocalConfig } = context.input

  const { offchainConfig } = await serializeOffchainConfig(offchainLocalConfig, process.env.SECRET!, randomSecret)
  const localConfig = offchainConfig.toString('base64')

  const proposal: any = await context.provider.wasm.contractQuery(context.contract, {
    proposal: {
      id: proposalId,
    },
  })

  try {
    assert.equal(localConfig, proposal.offchain_config)
  } catch (err) {
    throw new Error(`RDD configuration does not correspond to the proposal configuration. Error: ${err.message}`)
  }
  logger.success('RDD Generated configuration matches with onchain proposal configuration')

  // Config in Proposal
  const offchainConfigInProposal = deserializeConfig(Buffer.from(proposal.offchain_config, 'base64'))
  const configInProposal = prepareOffchainConfigForDiff(offchainConfigInProposal, { f: proposal.f })

  // Config in contract
  const event = await getLatestOCRConfigEvent(context.provider, context.contract)
  const offchainConfigInContract = event?.offchain_config
    ? deserializeConfig(Buffer.from(event.offchain_config[0], 'base64'))
    : ({} as OffchainConfig)
  const configInContract = prepareOffchainConfigForDiff(offchainConfigInContract, { f: event?.f[0] })

  logger.info('Review the configuration difference from contract and proposal: green - added, red - deleted.')
  diff.printDiff(configInContract, configInProposal)
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    id: input.proposalId,
    digest: Buffer.from(input.digest, 'hex').toString('base64'),
  }
}

const validateInput = (input: CommandInput): boolean => {
  if (!input.proposalId) throw new Error('A Config Proposal ID is required. Provide it with --configProposal flag')
  if (!input.randomSecret)
    throw new Error('Secret generated at proposing offchain config is required. Provide it with --secret flag')
  return true
}

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

const instruction: AbstractInstruction<CommandInput, ContractInput> = {
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
}

export default instructionToCommand(instruction)
