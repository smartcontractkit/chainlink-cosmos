import { CATEGORIES } from '../../../lib/constants'
import { isValidAddress } from '../../../lib/utils'
import { AbstractInstruction, instructionToCommand } from '../../abstract/executionWrapper'

type CommandInput = {
  owners: string[]
  admin?: string
}

type ContractInput = {
  members: {
    addr: string
    weight: number
  }[]
  admin?: string
}

const makeCommandInput = async (flags: any, args: any[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {
    owners: args,
    admin: flags.admin,
  }
}
const validateInput = (input: CommandInput): boolean => {
  const areValidOwners = input.owners.filter((owner) => !isValidAddress(owner)).length === 0
  if (!areValidOwners) throw new Error('Owners are not valid')
  if (input.admin && !isValidAddress(input.admin)) throw new Error('Admin is not valid')
  return true
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    members: input.owners.map((owner) => ({
      addr: owner,
      // Same weight for every owner
      weight: 1,
    })),
    admin: input.admin,
  }
}

// yarn gauntlet cw4_group:deploy --network=bombay-testnet <OWNERS_LIST>
const createGroupInstruction: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.MULTISIG,
    contract: 'cw4_group',
    function: 'deploy',
  },
  makeInput: makeCommandInput,
  validateInput,
  makeContractInput,
}

export const CreateGroup = instructionToCommand(createGroupInstruction)
