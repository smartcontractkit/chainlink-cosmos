import { CATEGORIES } from '../../../lib/constants'
import { AbstractInstruction, instructionToCommand } from '../../abstract/executionWrapper'
import { Duration, validDuration, Threshold, validThreshold, GroupMember, validAddr } from './lib/types'

type CreateGroupInput = {
  members : string[],
  weights : number[],
  admin?: string
}

type GroupInitParams = {
  members: GroupMember[],
  admin?: string
}

const makeCreateGroupInput = async (flags: any): Promise<CreateGroupInput> => {
  if (flags.input)
    return flags.input as CreateGroupInput

  return {
    members: flags.members,
    weights: flags.weights,
    admin: flags.admin
  }
}

// TODO: Add validation
const validateCreateGroupInput = (input: CreateGroupInput): boolean => {
  return true
}

const makeGroupInitParams = async (input : CreateGroupInput): Promise<GroupInitParams> => {
  return {
    members: input.members.map((a, i) => ({
        addr: a,
        weight: input.weights[i]
      })
    ),
    admin: input.admin
  }
}

type WalletInitParams = {
  group_addr: string,
  max_voting_period: Duration,
  threshold: Threshold
}

const makeWalletInitParams = async (flags: any):Promise<WalletInitParams> => {
  if (flags.input) return flags.input as WalletInitParams
  return {
    group_addr: flags.group_addr,
    max_voting_period: flags.max_voting_period,
    threshold: flags.threshold
  }
}

const validateWalletInitParams = (params: WalletInitParams): boolean => {
   let group_addr: string = params.group_addr;
   let max_voting_period: Duration = params.max_voting_period;
   let threshold: Threshold = params.threshold;

   if (!validAddr(group_addr)) {
    console.log(`group_addr=${group_addr} is not a valid terra address`)
    return false;
  }

  if (!validDuration(params.max_voting_period)) {
    console.log(`max_voting_period=${max_voting_period} is not a valid Duration`)
    return false;
  }
  
  if (!validThreshold(params.threshold)) {
    console.log(`threshold=${threshold} is not a valid Threshold`)
    return false;
  }
  return true
}

const makeContractInput = async (params: WalletInitParams): Promise<WalletInitParams> => {
  return {
    group_addr: params.group_addr,
    max_voting_period: params.max_voting_period,
    threshold: params.threshold
  }
}

const createGroupInstruction:  AbstractInstruction<CreateGroupInput, GroupInitParams> = {
  instruction: {
    category: CATEGORIES.MULTISIG,
    contract: 'cw4_group',
    function: 'deploy'
  },
  makeInput: makeCreateGroupInput,
  validateInput: validateCreateGroupInput,
  makeContractInput: makeGroupInitParams
}

// Creates a multisig wallet backed by a previously created cw4_group
const createWalletInstruction: AbstractInstruction<WalletInitParams, WalletInitParams> = {
  instruction: {
    category: CATEGORIES.MULTISIG,
    contract: 'cw3_flex_multisig',
    function: 'deploy'
  },
  makeInput: makeWalletInitParams,
  validateInput: validateWalletInitParams,
  makeContractInput: makeContractInput
}

export const CreateGroup = instructionToCommand(createGroupInstruction)
export const CreateWallet = instructionToCommand(createWalletInstruction)
