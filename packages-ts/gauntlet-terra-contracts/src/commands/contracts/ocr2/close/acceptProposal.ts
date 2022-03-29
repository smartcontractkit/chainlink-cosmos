import { Result } from '@chainlink/gauntlet-core'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TransactionResponse } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../../../lib/constants'
import { AbstractInstruction, instructionToCommand, BeforeExecute } from '../../../abstract/executionWrapper'
import { EMPTY_CONFIG } from './proposeOffchainConfig'

type CommandInput = {
  proposalId: string
  digest: string
}

type ContractInput = {
  id: string
  digest: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {
    proposalId: flags.proposalId || flags.configProposal, // --configProposal alias requested by eng ops
    digest: flags.digest,
  }
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context, inputContext) => async () => {
  const proposal: any = await context.provider.wasm.contractQuery(context.contract, {
    proposal: {
      id: inputContext.contractInput.id,
    },
  })

  if (proposal.offchain_config !== EMPTY_CONFIG) {
    throw new Error('You are accepting a proposal with a non empty config')
  }

  logger.line()
  logger.info('IMPORTANT')
  logger.info('You are accepting a proposal for CLOSING a feed. Run this command only if you know what you are doing!')
  logger.line()
  await prompt('Continue?')
  await prompt('Are you sure you want to continue?')
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    id: input.proposalId,
    digest: Buffer.from(input.digest, 'hex').toString('base64'),
  }
}

const validateInput = (input: CommandInput): boolean => {
  if (!input.proposalId) throw new Error('A Config Proposal ID is required. Provide it with --configProposal flag')
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
    'yarn gauntlet ocr2:accept_proposal:close --network=<NETWORK> --configProposal=<PROPOSAL_ID> --digest=<DIGEST> <CONTRACT_ADDRESS>',
  ],
  instruction: {
    category: CATEGORIES.OCR,
    contract: 'ocr2',
    function: 'accept_proposal',
    subInstruction: 'close',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  beforeExecute,
  afterExecute,
}

export default instructionToCommand(instruction)
