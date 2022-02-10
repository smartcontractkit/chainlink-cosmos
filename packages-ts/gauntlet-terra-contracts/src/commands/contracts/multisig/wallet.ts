import { CATEGORIES } from '../../../lib/constants'
import { AbstractInstruction, instructionToCommand } from '../../abstract/wrapper'
import { Duration, isValidDuration, Threshold, isValidThreshold } from './lib/types'
import { isValidAddress } from '../../../lib/schema'

type WalletInitParams = {
  group_addr: string
  max_voting_period: Duration
  threshold: Threshold
}

const makeWalletInitParams = async (flags: any): Promise<WalletInitParams> => {
  if (flags.input) return flags.input as WalletInitParams
  return {
    group_addr: flags.group_addr,
    max_voting_period: flags.max_voting_period,
    threshold: flags.threshold,
  }
}

const validateWalletInitParams = (params: WalletInitParams): boolean => {
  if (!isValidAddress(params.group_addr)) {
    console.log(`group_addr=${params.group_addr} is not a valid terra address`)
    return false
  }

  if (!isValidDuration(params.max_voting_period)) {
    console.log(`max_voting_period=${params.max_voting_period} is not a valid Duration`)
    return false
  }

  if (!isValidThreshold(params.threshold)) {
    console.log(`threshold=${params.threshold} is not a valid Threshold`)
    return false
  }

  return true
}

const makeContractInput = async (params: WalletInitParams): Promise<WalletInitParams> => {
  return {
    group_addr: params.group_addr,
    max_voting_period: params.max_voting_period,
    threshold: params.threshold,
  }
}

// Creates a multisig wallet backed by a previously created cw4_group
const createWalletInstruction: AbstractInstruction<WalletInitParams, WalletInitParams> = {
  instruction: {
    category: CATEGORIES.MULTISIG,
    contract: 'cw3_flex_multisig',
    function: 'deploy',
  },
  makeInput: makeWalletInitParams,
  validateInput: validateWalletInitParams,
  makeContractInput: makeContractInput,
}

export const CreateWallet = instructionToCommand(createWalletInstruction)
