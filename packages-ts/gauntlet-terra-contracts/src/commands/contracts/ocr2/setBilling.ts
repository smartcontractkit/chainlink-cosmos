import { getRDD } from '../../../lib/rdd'
import { abstractWrapper } from '../../abstract/wrapper'

type CommandInput = {
  recommendedGasPrice: number
  observationPaymentGjuels: number
  transmissionPaymentGjuels: number
}

type ContractInput = {
  config: {
    recommended_gas_price: number
    observation_payment: number
  }
}

export const makeCommandInput = async (flags: any): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  const rdd = getRDD(flags.rdd)
  const billingInfo = rdd.contracts[flags.contract]?.billing
  return {
    observationPaymentGjuels: billingInfo.observationPaymentGjuels,
    transmissionPaymentGjuels: billingInfo.transmissionPaymentGjuels,
    recommendedGasPrice: billingInfo.recommendedGasPrice,
  }
}

export const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    config: {
      recommended_gas_price: input.recommendedGasPrice,
      observation_payment: input.observationPaymentGjuels,
    },
  }
}

// TODO: Add validation
export const validateInput = (input: CommandInput): boolean => {
  return true
}

export const makeOCR2SetBillingCommand = (flags: any, args: string[]) => {
  return abstractWrapper<CommandInput, ContractInput>(
    {
      instruction: 'ocr2:set_billing',
      flags,
      contract: args[0],
    },
    makeCommandInput,
    makeContractInput,
    validateInput,
  )
}
