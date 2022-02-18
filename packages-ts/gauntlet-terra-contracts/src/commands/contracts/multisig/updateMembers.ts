import { CATEGORIES } from '../../../lib/constants'
import { Member } from '../../../lib/multisig'
import { isValidAddress } from '../../../lib/utils'
import { AbstractInstruction, instructionToCommand } from '../../abstract/executionWrapper'

type CommandInput = {
  add?: string[]
  remove?: string[]
}

type ContractInput = {
  add: Member[]
  remove: string[]
}

const makeCommandInput = async (flags: any, args: any[]): Promise<CommandInput> => {
  return {
    add: flags.add?.split(','),
    remove: flags.remove?.split(','),
  } as CommandInput
}

const validateInput = (input: CommandInput): boolean => {
  if (input.add && !input.add?.every((addr) => isValidAddress(addr))) {
    throw new Error("One of provided 'add' addresses is not valid!")
  }

  if (input.remove && !input.remove?.every((addr) => isValidAddress(addr))) {
    throw new Error("One of provided 'remove' addresses of not valid!")
  }

  if (!input.add && !input.remove) {
    throw new Error("You must specify 'add' or 'remove' addresses!")
  }

  return true
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  const membersToAdd = input.add?.map((addr: string) => {
    return {
      addr,
      weight: 1,
    } as Member
  })

  return {
    add: membersToAdd || [],
    remove: input.remove || [],
  } as ContractInput
}

// yarn gauntlet cw4_group:update_members --add=<ADDRESS1_TO_ADD>,<ADDRESS2_TO_ADD> --remove=<ADDRESS3_TO_REMOVE> <CONTRACT_ADDRESS>
// either --remove and --add can be omitted
const createUpdateMembersInstruction: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.MULTISIG,
    contract: 'cw4_group',
    function: 'update_members',
  },
  makeInput: makeCommandInput,
  validateInput,
  makeContractInput,
}

export const UpdateMembers = instructionToCommand(createUpdateMembersInstruction)
