import { extendCommandInstruction, instructionToCommand } from '../../../abstract/executionWrapper'
import ProposeOffchainConfig, { CommandInput, OffchainConfig, instruction } from '../proposeOffchainConfig'

export const makeEmptyOffchainConfig = (): OffchainConfig => {
  return ({
    offchainPublicKeys: [],
    configPublicKeys: [],
    peerIds: [],
  } as unknown) as OffchainConfig
}

export const EMPTY_SECRET = 'EMPTY'

const makeInput = async (flags): Promise<CommandInput> => {
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
    proposalId: flags.configProposal,
    ...defaultInput,
  }
}

export default instructionToCommand(
  extendCommandInstruction(instruction, {
    suffixes: ['close'],
    examples: [`yarn gauntlet ${ProposeOffchainConfig.id}:close --network=<NETWORK> <CONTRACT_ADDRESS>`],
    makeInput,
    validationsToSkip: [0],
  }),
)
