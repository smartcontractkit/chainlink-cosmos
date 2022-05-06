import { Result } from '@chainlink/gauntlet-core'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { TransactionResponse } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../../../lib/constants'
import { AbstractInstruction, instructionToCommand } from '../../../abstract/executionWrapper'

type CommandInput = {
  proposalId: string
}

type ContractInput = {
  id: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {
    proposalId: flags.proposalId || flags.configProposal || flags.id, // -configProposal alias requested by eng ops
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    id: input.proposalId,
  }
}

const validateInput = (input: CommandInput): boolean => {
  if (!input.proposalId) throw new Error('A Config Proposal ID is required. Provide it with --configProposal flag')
  return true
}

const afterExecute = () => async (
  response: Result<TransactionResponse>,
): Promise<{ proposalId: string; digest: string } | undefined> => {
  const events = response.responses[0].tx.events
  if (!events) {
    logger.error('Could not retrieve events from tx')
    return
  }

  const proposalId = events[0].wasm.proposal_id[0]
  const digest = events[0].wasm.digest[0]
  logger.success(`Config Proposal ${proposalId} finalized`)
  logger.line()
  logger.info('Important: Save the config proposal DIGEST to accept the proposal in the future:')
  logger.info(digest)
  logger.line()
  return {
    proposalId,
    digest,
  }
}

const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  examples: [
    'yarn gauntlet ocr2:finalize_proposal --network=<NETWORK> --configProposal=<PROPOSAL_ID> <CONTRACT_ADDRESS>',
  ],
  instruction: {
    contract: 'ocr2',
    function: 'finalize_proposal',
    category: CATEGORIES.OCR,
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  afterExecute,
}

export default instructionToCommand(instruction)
