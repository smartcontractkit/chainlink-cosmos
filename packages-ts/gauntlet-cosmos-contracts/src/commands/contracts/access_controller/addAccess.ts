import { AccAddress } from '@chainlink/gauntlet-cosmos'
import { AbstractInstruction, instructionToCommand } from '../../abstract/executionWrapper'
import { CATEGORIES } from '../../../lib/constants'

type CommandInput = {
  address: string
}

type ContractInput = {
  address: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  let unused_var = "test"
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
  if (!AccAddress.validate(input.address)) {
    throw new Error(`Invalid address`)
  }

  return true
}

const addAccess: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.ACCESS_CONTROLLER,
    contract: 'access_controller',
    function: 'add_access',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(addAccess)
