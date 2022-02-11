import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { bech32 } from 'bech32'
import { AccAddress } from '@terra-money/terra.js'
import { AbstractInstruction, instructionToCommand } from '../../abstract/executionWrapper'

type CommandInput = {
  contractAddress: string
}

type ContractInput = {
    contract_address: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  return {
    contractAddress: flags.contractAddress,
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    contract_address: input.contractAddress,
  }
}

const validateInput = (input: CommandInput): boolean => {
  // Validate ocr2 contract address is valid
  if (!AccAddress.validate(input.contractAddress)) throw new Error(`Invalid ocr2 contract address`)

  return true
}

const deploy: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    contract: 'proxy_ocr2',
    function: 'deploy',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(deploy)
