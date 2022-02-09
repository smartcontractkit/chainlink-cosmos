import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { AbstractInstruction, instructionToCommand } from '../../abstract/wrapper'

type CommandInput = {
  address: String
}

type ContractInput = {
  address: String
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  return {
    address: flags.address,
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    address: input.address,
  }
}

const validateInput = (input: CommandInput): boolean => {
  const { prefix: decodedPrefix } = bech32.decode(input.address) // throws error if checksum is invalid which will fail validation

  // verify address prefix
  if (decodedPrefix !== 'terra') {
    throw new Error(`Invalid address prefix (expecteed: 'terra')`)
  }

  return true
}

const addAccess: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    contract: 'access_controller',
    function: 'add_access',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(addAccess)
