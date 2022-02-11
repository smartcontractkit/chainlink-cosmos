import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { bech32 } from 'bech32'
import { AccAddress } from '@terra-money/terra.js'
import { AbstractInstruction, instructionToCommand } from '../../abstract/executionWrapper'

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
  // Validate ocr2 contract address is valid
  if (!AccAddress.validate(input.address)) throw new Error(`Invalid ocr2 contract address`)

  return true
}

const confirmContract: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    contract: 'proxy_ocr2',
    function: 'confirm_contract',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(confirmContract)
