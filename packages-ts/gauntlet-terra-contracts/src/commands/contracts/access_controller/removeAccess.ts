import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { AccAddress } from '@terra-money/terra.js'
import { abstract } from '../..'
import {
  AbstractInstruction,
  instructionToCommand,
} from '@chainlink/gauntlet-terra/dist/commands/abstract/executionWrapper'
import { CATEGORIES } from '../../../lib/constants'
import { CONTRACT_LIST, getContract } from '../../../lib/contracts'

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

const removeAccess: AbstractInstruction<CommandInput, ContractInput, CONTRACT_LIST> = {
  instruction: {
    category: CATEGORIES.ACCESS_CONTROLLER,
    contract: 'access_controller',
    function: 'remove_access',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  getContract: getContract,
}

export default abstract.instructionToCommand(removeAccess)
