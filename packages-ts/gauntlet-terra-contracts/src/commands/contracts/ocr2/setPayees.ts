import { getRDD } from '../../../lib/rdd'
import { AbstractInstruction, instructionToCommand } from '../../abstract/wrapper'

type CommandInput = {
  payees: string[]
  transmitters: string[]
}

type ContractInput = {
  payees: string[][]
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  const rdd = getRDD(flags.rdd)
  const contract = args[0]
  const aggregator = rdd.contracts[contract]
  const aggregatorOperators: string[] = aggregator.oracles.map((o) => o.operator)
  const payees = aggregatorOperators.map((operator) => rdd.operators[operator].adminAddress)
  const transmitters = aggregatorOperators.map((operator) => rdd.operators[operator].ocrNodeAddress[0])
  return {
    payees,
    transmitters,
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    payees: input.payees.map((payee, i) => [payee, input.transmitters[i]]),
  }
}

// TODO: Add validation
const validateInput = (input: CommandInput): boolean => {
  return true
}

const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    contract: 'ocr2',
    function: 'set_payees',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(instruction)
