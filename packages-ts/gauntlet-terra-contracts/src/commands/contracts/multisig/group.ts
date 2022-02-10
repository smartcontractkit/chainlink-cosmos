import { CATEGORIES } from '../../../lib/constants'
import { AbstractInstruction, instructionToCommand } from '../../abstract/wrapper'
import { GroupMember } from './lib/types'

type CreateGroupInput = {
  members: string[]
  weights: number[]
  admin?: string
}

type GroupInitParams = {
  members: GroupMember[]
  admin?: string
}

const makeCreateGroupInput = async (flags: any): Promise<CreateGroupInput> => {
  if (flags.input) return flags.input as CreateGroupInput

  return {
    members: flags.members,
    weights: flags.weights,
    admin: flags.admin,
  }
}

// TODO: Add validation
const validateCreateGroupInput = (input: CreateGroupInput): boolean => {
  return true
}

const makeGroupInitParams = async (input: CreateGroupInput): Promise<GroupInitParams> => {
  return {
    members: input.members.map((a, i) => ({
      addr: a,
      weight: input.weights[i],
    })),
    admin: input.admin,
  }
}

const createGroupInstruction: AbstractInstruction<CreateGroupInput, GroupInitParams> = {
  instruction: {
    category: CATEGORIES.MULTISIG,
    contract: 'cw4_group',
    function: 'deploy',
  },
  makeInput: makeCreateGroupInput,
  validateInput: validateCreateGroupInput,
  makeContractInput: makeGroupInitParams,
}

export const CreateGroup = instructionToCommand(createGroupInstruction)
