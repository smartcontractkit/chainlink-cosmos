import { instructionToTailCommand, TailInstruction } from '../../../abstract/executionWrapper'
import AcceptProposal, { CommandInput as AcceptProposalInput } from '../proposal/acceptProposal'
import { EMPTY_SECRET, makeEmptyOffchainConfig } from './proposeOffchainConfig'

type CommandInput = Pick<AcceptProposalInput, 'proposalId' | 'digest'>

const makeInput = (flags, args): CommandInput => {
  if (flags.input) return flags.input as CommandInput

  return {
    proposalId: flags.configProposal,
    digest: flags.digest,
  }
}

const makeInnerCommandInput = (input: CommandInput): AcceptProposalInput => {
  return {
    proposalId: input.proposalId,
    digest: input.digest,
    offchainConfig: makeEmptyOffchainConfig(),
    randomSecret: EMPTY_SECRET,
  }
}

const instruction: TailInstruction<CommandInput, AcceptProposalInput> = {
  command: AcceptProposal,
  ui: {
    suffixes: ['close'],
    examples: [`yarn gauntlet ${AcceptProposal.id}:close --network=<NETWORK> <CONTRACT_ADDRESS>`],
  },
  makeInput,
  makeInnerCommandInput,
  skipValidations: [],
}

export default instructionToTailCommand(instruction)
