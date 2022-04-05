import { MnemonicKey } from '@terra-money/terra.js'
import { extendCommandInstruction, instructionToCommand } from '../../../abstract/executionWrapper'
import ProposeConfig, { CommandInput, instruction } from '../proposeConfig'

const makeInput = async (flags): Promise<CommandInput> => {
  const randomAcc = () => new MnemonicKey().publicKey?.address()!
  const makeEmptyOracle = (n: number) => ({
    signer: new Array(64).fill(n.toString(16)).join(''),
    transmitter: randomAcc(),
    payee: randomAcc(),
  })
  // > f * 3 oracles
  const oracles = new Array(4).fill('').map((_, i) => makeEmptyOracle(i))
  const defaultInput = {
    f: Number(1),
    onchainConfig: '',
    signers: oracles.map((o) => o.signer),
    transmitters: oracles.map((o) => o.transmitter),
    payees: oracles.map((o) => o.payee),
  }

  if (flags.input) {
    return {
      ...flags.input,
      ...defaultInput,
    }
  }

  return {
    proposalId: flags.configProposal,
    ...defaultInput,
  }
}

export default instructionToCommand(
  extendCommandInstruction(instruction, {
    suffixes: ['close'],
    examples: [`yarn gauntlet ${ProposeConfig.id}:close --network=<NETWORK> <CONTRACT_ADDRESS>`],
    makeInput,
  }),
)
