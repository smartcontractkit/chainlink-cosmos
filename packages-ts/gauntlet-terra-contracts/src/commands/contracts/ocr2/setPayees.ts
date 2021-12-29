import { getRDD } from '../../../lib/rdd'
import { abstractWrapper } from '../../abstract/wrapper'

type CommandInput = {
  payees: string[]
  transmitters: string[]
}

type ContractInput = {
  payees: string[][]
}

export const makeCommandInput = async (flags: any): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  const rdd = getRDD(flags.rdd)
  const aggregator = rdd.contracts[flags.contract]
  const aggregatorOperators: string[] = aggregator.oracles.map((o) => o.operator)
  const payees = aggregatorOperators.map((operator) => rdd.operators[operator].adminAddress)
  const transmitters = aggregatorOperators.map((operator) => rdd.operators[operator].ocrNodeAddress[0])
  return {
    payees,
    transmitters,
  }
}

export const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    payees: input.payees.map((payee, i) => [payee, input.transmitters[i]]),
  }
}

// TODO: Add validation
export const validateInput = (input: CommandInput): boolean => {
  return true
}

export const makeOCR2SetPayeesCommand = (flags: any, args: string[]) => {
  return abstractWrapper<CommandInput, ContractInput>(
    {
      instruction: 'ocr2:set_payees',
      flags,
      contract: args[0],
    },
    makeCommandInput,
    makeContractInput,
    validateInput,
  )
}
