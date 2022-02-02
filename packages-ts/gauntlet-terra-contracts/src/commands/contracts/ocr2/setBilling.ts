import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { getRDD } from '../../../lib/rdd'
import { AbstractInstruction, instructionToCommand } from '../../abstract/wrapper'

type CommandInput = {
  recommendedGasPrice: number
  observationPaymentGjuels: number
  transmissionPaymentGjuels: number
}

type ContractInput = {
  config: {
    recommended_gas_price: number
    observation_payment_gjuels: number
    transmission_payment_gjuels: number
  }
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  const rdd = getRDD(flags.rdd)
  const contract = args[0]
  const billingInfo = rdd.contracts[contract]?.billing
  return {
    observationPaymentGjuels: billingInfo.observationPaymentGjuels,
    transmissionPaymentGjuels: billingInfo.transmissionPaymentGjuels,
    recommendedGasPrice: billingInfo.recommendedGasPrice,
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    config: {
      observation_payment_gjuels: new BN(input.observationPaymentGjuels).toNumber(),
      transmission_payment_gjuels: new BN(input.transmissionPaymentGjuels).toNumber(),
      recommended_gas_price: new BN(input.recommendedGasPrice).toNumber(),
    },
  }
}

// TODO: Add validation
const validateInput = (input: CommandInput): boolean => {
  return true
}

const setBillingInstruction: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    contract: 'ocr2',
    function: 'set_billing',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(setBillingInstruction)
