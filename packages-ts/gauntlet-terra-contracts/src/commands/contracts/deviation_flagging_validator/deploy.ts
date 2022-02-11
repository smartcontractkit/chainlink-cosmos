import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { AccAddress } from '@terra-money/terra.js'
import { AbstractInstruction, instructionToCommand } from '../../abstract/executionWrapper'

type CommandInput = {
  address: string
  flaggingThreshold: number
}

type ContractInput = {
  flags: string
  flagging_threshold: number
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  return {
    address: flags.flags,
    flaggingThreshold: flags.flaggingThreshold,
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    flags: input.address,
    flagging_threshold: Number(input.flaggingThreshold),
  }
}

const validateInput = (input: CommandInput): boolean => {
  // Validate flags contract address is valid
  if (!AccAddress.validate(input.address)) throw new Error(`Invalid flags address`)

  // Flagging threshold must be greater than 0
  if (input.flaggingThreshold <= 0) throw new Error(`Flagging threshold must be greater than 0`)

  return true
}

const deploy: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    contract: 'deviation_flagging_validator',
    function: 'deploy',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(deploy)
