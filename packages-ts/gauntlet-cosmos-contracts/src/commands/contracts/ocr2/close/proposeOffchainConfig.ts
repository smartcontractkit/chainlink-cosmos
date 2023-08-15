import { encoding } from '@chainlink/gauntlet-contracts-ocr2'
import { extendCommandInstruction, instructionToCommand } from '../../../abstract/executionWrapper'
import ProposeOffchainConfig, { CommandInput, instruction } from '../proposeOffchainConfig'

export const makeEmptyOffchainConfig = (): encoding.OffchainConfig => {
  return {
    offchainPublicKeys: [],
    configPublicKeys: [],
    peerIds: [],
  } as unknown as encoding.OffchainConfig
}

export const EMPTY_SECRET = 'EMPTY'

const makeInput = async (flags): Promise<CommandInput> => {
  const defaultInput = {
    f: 0,
    signers: [],
    transmitters: [],
    onchainConfig: [],
    offchainConfig: makeEmptyOffchainConfig(),
    offchainConfigVersion: 0,
    secret: EMPTY_SECRET,
    randomSecret: EMPTY_SECRET,
  }
  if (flags.input) return { ...flags.input, ...defaultInput }

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
    validationsToSkip: ['validOffchainConfig'],
  }),
)
