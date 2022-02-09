import { CATEGORIES } from '../../../lib/constants'
import { isValidAddress } from '../../../lib/utils'
import { AbstractInstruction, instructionToCommand } from '../../abstract/wrapper'

type Duration = {
  height?: number // block height
  time?: number // length of time in seconds
}

type CommandInput = {
  group: string
  threshold: number
  votingPeriod?: Duration
}

type ContractInput = {
  group_addr: string
  max_voting_period: Duration
  threshold: Threshold
}

type Threshold = {
  absolute_count?: {
    weight: number
  }
  absolute_percentage?: {
    percentage: number
  }
  threshold_quorum?: {
    threshold: number
    quorum: number
  }
}

const makeCommandInput = async (flags: any): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {
    group: flags.group,
    threshold: Number(flags.threshold),
    votingPeriod: {
      height: flags.height,
      time: flags.time,
    },
  }
}

const validateInput = (input: CommandInput): boolean => {
  // TODO: Add time validation
  const isValidTime = (a: any) => true
  if (!isValidAddress(input.group)) {
    throw new Error(`group ${input.group} is not a valid terra address`)
  }

  if (input.threshold === 0) {
    throw new Error(`Threshold ${input.threshold} is not a valid. Should be higher than zero`)
  }

  if (input.votingPeriod?.height && isNaN(input.votingPeriod?.height)) {
    throw new Error(`Voting period height ${input.votingPeriod.height} is not a valid Block`)
  }

  if (input.votingPeriod?.time && !isValidTime(input.votingPeriod?.time)) {
    throw new Error(`Voting period time ${input.votingPeriod?.time} is not a valid time`)
  }

  return true
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    group_addr: input.group,
    max_voting_period: {
      height: input.votingPeriod?.height,
      time: input.votingPeriod?.time,
    },
    threshold: {
      absolute_count: {
        weight: input.threshold,
      },
    },
  }
}

// Creates a multisig wallet backed by a previously created cw4_group
const createWalletInstruction: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.MULTISIG,
    contract: 'cw3_flex_multisig',
    function: 'deploy',
  },
  makeInput: makeCommandInput,
  validateInput,
  makeContractInput: makeContractInput,
}

export const CreateWallet = instructionToCommand(createWalletInstruction)
