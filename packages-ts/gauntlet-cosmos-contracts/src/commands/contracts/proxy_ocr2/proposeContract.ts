import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { AccAddress } from '@chainlink/gauntlet-cosmos'
import { CATEGORIES } from '../../../lib/constants'
import { CONTRACT_LIST } from '../../../lib/contracts'
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

const proposeContract: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.PROXIES,
    contract: CONTRACT_LIST.PROXY_OCR_2,
    function: 'propose_contract',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(proposeContract)
