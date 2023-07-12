import { AccAddress } from '@chainlink/gauntlet-cosmos'
import { CATEGORIES } from '../../../lib/constants'
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
  if (!AccAddress.validate(input.admin)) {
    throw new Error('Admin address is not valid!')
  }

  return true
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    admin: input.admin,
  } as ContractInput
}

const createUpdateAdminInstruction: AbstractInstruction<CommandInput, ContractInput> = {
  examples: ['yarn gauntlet cw4_group:update_admin --admin=<NEW_ADMIN_ADDRESS> <CONTRACT_ADDRESS>'],
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
