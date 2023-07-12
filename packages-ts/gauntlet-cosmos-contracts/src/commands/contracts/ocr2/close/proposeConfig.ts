import { AccAddress } from '@chainlink/gauntlet-cosmos'
import { EMPTY_TRANSMITTERS } from '../../../../lib/constants'
import { extendCommandInstruction, instructionToCommand } from '../../../abstract/executionWrapper'
import ProposeConfig, { CommandInput, instruction } from '../proposeConfig'

const makeInput = async (flags): Promise<CommandInput> => {
  const makeEmptyOracle = (n: number, emptyAddress: AccAddress) => ({
    signer: new Array(64).fill(n.toString(16)).join(''),
    transmitter: emptyAddress,
    payee: emptyAddress,
  })
  // > f * 3 oracles
  const oracles = new Array(4).fill('').map((_, i) => makeEmptyOracle(i, EMPTY_TRANSMITTERS[i]))
  const defaultInput = {
    f: Number(1),
    onchainConfig: '',
    signers: oracles.map((o) => o.signer),
    transmitters: oracles.map((o) => o.transmitter),
    payees: oracles.map((o) => o.payee),
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
    examples: [`yarn gauntlet ${ProposeConfig.id}:close --network=<NETWORK> <CONTRACT_ADDRESS>`],
    makeInput,
  }),
)
