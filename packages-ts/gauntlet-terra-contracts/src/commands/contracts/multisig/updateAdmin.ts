import { CATEGORIES } from '../../../lib/constants'
import { isValidAddress } from '../../../lib/utils'
import { AbstractInstruction, instructionToCommand } from '../../abstract/executionWrapper'

type CommandInput = {
  admin: string
}

type ContractInput = {
  admin: string
}

const makeCommandInput = async (flags: any, args: any[]): Promise<CommandInput> => {
  return {
    admin: flags.admin,
  } as CommandInput
}

const validateInput = (input: CommandInput): boolean => {
  if (!isValidAddress(input.admin)) {
    throw new Error('Admin address is not valid!')
  }

  return true
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    admin: input.admin,
  } as ContractInput
}

// yarn gauntlet cw4_group:update_admin --admin=<NEW_ADMIN_ADDRESS> <CONTRACT_ADDRESS>
const createUpdateAdminInstruction: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.MULTISIG,
    contract: 'cw4_group',
    function: 'update_admin',
  },
  makeInput: makeCommandInput,
  validateInput,
  makeContractInput,
}

export const UpdateAdmin = instructionToCommand(createUpdateAdminInstruction)
