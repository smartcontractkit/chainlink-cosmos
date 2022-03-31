import { instructionToTailCommand, TailInstruction } from '../../../abstract/executionWrapper'
import ProposeOffchainConfig, { CommandInput as ProposeOffchainConfigInput } from '../proposeOffchainConfig'
import { OffchainConfig } from '../proposeOffchainConfig'

type CommandInput = Pick<ProposeOffchainConfigInput, 'proposalId'>

export const makeEmptyOffchainConfig = (): OffchainConfig => {
  return ({
    offchainPublicKeys: [],
    configPublicKeys: [],
    peerIds: [],
  } as unknown) as OffchainConfig
}

export const EMPTY_SECRET = 'EMPTY'

const makeInput = (flags, args): CommandInput => {
  if (flags.input) return flags.input as CommandInput

  return {
    proposalId: flags.configProposal,
  }
}

const makeInnerCommandInput = (input: CommandInput): ProposeOffchainConfigInput => {
  return {
    proposalId: input.proposalId,
    offchainConfig: makeEmptyOffchainConfig(),
    randomSecret: EMPTY_SECRET,
  }
}

const instruction: TailInstruction<CommandInput, ProposeOffchainConfigInput> = {
  command: ProposeOffchainConfig,
  ui: {
    suffixes: ['close'],
    examples: [`yarn gauntlet ${ProposeOffchainConfig.id}:close --network=<NETWORK> <CONTRACT_ADDRESS>`],
  },
  makeInput,
  makeInnerCommandInput,
  skipValidations: ['OCRConfig'],
}

export default instructionToTailCommand(instruction)
