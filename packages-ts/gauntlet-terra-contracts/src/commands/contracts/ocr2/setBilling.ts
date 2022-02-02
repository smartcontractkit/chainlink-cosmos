import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { getRDD } from '../../../lib/rdd'
import { AbstractInstruction, instructionToCommand } from '../../abstract/wrapper'

type CommandInput = {
  recommendedGasPriceUluna: string
  observationPaymentGjuels: number
  transmissionPaymentGjuels: number
}

type ContractInput = {
  config: {
    recommended_gas_price_uluna: string
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
    recommendedGasPriceUluna: billingInfo.recommendedGasPriceUluna,
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    config: {
      observation_payment_gjuels: new BN(input.observationPaymentGjuels).toNumber(),
      transmission_payment_gjuels: new BN(input.transmissionPaymentGjuels).toNumber(),
      recommended_gas_price_uluna: new BN(input.recommendedGasPriceUluna).toString(),
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
