import { extendCommandInstruction, instructionToCommand } from '../../../abstract/executionWrapper'
import AcceptProposal, {
  CommandInput as AcceptProposalInput,
  instruction as acceptProposalInstruction,
} from '../proposal/acceptProposal'
import { EMPTY_SECRET, makeEmptyOffchainConfig } from './proposeOffchainConfig'

const makeInput = async (flags): Promise<AcceptProposalInput> => {
  const defaultInput = {
    offchainConfig: makeEmptyOffchainConfig(),
    randomSecret: EMPTY_SECRET,
  }
  if (flags.input)
    return {
      ...flags.input,
      ...defaultInput,
    }

  return {
    proposalId: flags.proposalId || flags.configProposal,
    digest: flags.digest,
    ...defaultInput,
  }
}

export default instructionToCommand(
  extendCommandInstruction(acceptProposalInstruction, {
    suffixes: ['close'],
    examples: [`yarn gauntlet ${AcceptProposal.id}:close --network=<NETWORK> <CONTRACT_ADDRESS>`],
    makeInput,
  }),
)
