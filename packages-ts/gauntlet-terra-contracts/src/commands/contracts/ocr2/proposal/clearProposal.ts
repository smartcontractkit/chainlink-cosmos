import { CATEGORIES } from '../../../../lib/constants'
import { instructionToCommand, AbstractInstruction } from '../../../abstract/executionWrapper'

type CommandInput = {
  proposalId: string
}

type ContractInput = {
  id: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {
    proposalId: flags.proposalId || flags.configProposal || flags.id, // --configProposal alias requested by eng ops
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

const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  examples: [
    'yarn gauntlet ocr2:clear_proposal --network=bombay-testnet --configProposal=<PROPOSAL_ID> <CONTRACT_ADDRESS>',
  ],
  instruction: {
    category: CATEGORIES.OCR,
    contract: 'ocr2',
    function: 'clear_proposal',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(instruction)
