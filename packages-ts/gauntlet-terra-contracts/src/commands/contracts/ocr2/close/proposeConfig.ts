import { MnemonicKey } from '@terra-money/terra.js'
import { instructionToTailCommand, TailInstruction } from '../../../abstract/executionWrapper'
import ProposeConfig, { CommandInput as ProposeConfigInput } from '../proposeConfig'

type CommandInput = Pick<ProposeConfigInput, 'proposalId'>

const makeInput = (flags, args): CommandInput => {
  if (flags.input) return flags.input as CommandInput
  return {
    proposalId: flags.configProposal,
  }
}

const makeInnerCommandInput = (input: CommandInput): ProposeConfigInput => {
  const randomAcc = () => new MnemonicKey().publicKey?.address()!
  const makeEmptyOracle = (n: number) => ({
    signer: new Array(64).fill(n.toString(16)).join(''),
    transmitter: randomAcc(),
    payee: randomAcc(),
  })
  // > f * 3 oracles
  const oracles = new Array(4).fill('').map((_, i) => makeEmptyOracle(i))

  return {
    proposalId: input.proposalId,
    f: Number(1),
    onchainConfig: '',
    signers: oracles.map((o) => o.signer),
    transmitters: oracles.map((o) => o.transmitter),
    payees: oracles.map((o) => o.payee),
  }
}

const instruction: TailInstruction<CommandInput, ProposeConfigInput> = {
  command: ProposeConfig,
  ui: {
    suffixes: ['close'],
    examples: [`yarn gauntlet ${ProposeConfig.id}:close --network=<NETWORK> <CONTRACT_ADDRESS>`],
  },
  makeInput,
  makeInnerCommandInput,
  skipValidations: [],
}

export default instructionToTailCommand(instruction)
