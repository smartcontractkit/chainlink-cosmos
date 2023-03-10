import { Result } from '@chainlink/gauntlet-core'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { TransactionResponse } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../../../lib/constants'
import { AbstractInstruction, instructionToCommand } from '../../../abstract/executionWrapper'

type CommandInput = {}

type ContractInput = {}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  return {}
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {}
}

const validateInput = (input: CommandInput): boolean => {
  return true
}

const afterExecute = (context) => async (
  response: Result<TransactionResponse>,
): Promise<{ proposalId: string } | undefined> => {
  const events = response.responses[0].tx.events
  if (!events) {
    logger.error('No events found. Config Proposal ID could not be retrieved')
    return
  }

  try {
    const proposalId = events.filter((element) => element.wasm[0].contract_address == context.contract)[0].wasm[0]
      .proposal_id

    if (!proposalId) {
      throw new Error('ProposalId for the given contract does not exist inside events')
    }

    logger.success(`New config proposal created on ${context.contract} with Config Proposal ID: ${proposalId}`)
    return {
      proposalId,
    }
  } catch (e) {
    logger.error('Config Proposal ID not found inside events')
    return
  }
}

const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  examples: ['yarn ocr2:begin_proposal --network=<NETWORK> <CONTRACT_ADDRESS>'],
  instruction: {
    contract: 'ocr2',
    function: 'begin_proposal',
    category: CATEGORIES.OCR,
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  afterExecute,
}

export default instructionToCommand(instruction)
