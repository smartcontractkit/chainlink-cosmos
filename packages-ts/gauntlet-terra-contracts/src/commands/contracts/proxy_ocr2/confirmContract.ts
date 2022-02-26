import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { AccAddress } from '@terra-money/terra.js'
import { abstract, AbstractInstruction } from '../..'
import { CATEGORIES } from '../../../lib/constants'
import { CONTRACT_LIST } from '../../../lib/contracts'

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
    category: CATEGORIES.PROXIES,
    contract: CONTRACT_LIST.PROXY_OCR_2,
    function: 'confirm_contract',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default abstract.instructionToCommand(confirmContract)
