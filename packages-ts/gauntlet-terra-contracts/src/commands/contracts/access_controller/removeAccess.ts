import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { AccAddress } from '@chainlink/gauntlet-terra'
import { AbstractInstruction, instructionToCommand } from '../../abstract/executionWrapper'
import { CATEGORIES } from '../../../lib/constants'
import { CONTRACT_LIST } from '../../../lib/contracts'

type CommandInput = {
  address: string
}

type ContractInput = {
  address: string
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
  if (!AccAddress.validate(input.address)) {
    throw new Error(`Invalid address`)
  }

  return true
}

const removeAccess: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.ACCESS_CONTROLLER,
    contract: 'access_controller',
    function: 'remove_access',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(removeAccess)
