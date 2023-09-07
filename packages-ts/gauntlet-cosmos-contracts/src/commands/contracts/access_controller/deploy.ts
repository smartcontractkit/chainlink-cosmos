import { AbstractInstruction, instructionToCommand } from '../../abstract/executionWrapper'
import { CATEGORIES } from '../../../lib/constants'

type CommandInput = {}

type ContractInput = {}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  return {}
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {}
}

const validateInput = (input: CommandInput): boolean => {
  return true
}

const deploy: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.ACCESS_CONTROLLER,
    contract: 'access_controller',
    function: 'deploy',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(deploy)
