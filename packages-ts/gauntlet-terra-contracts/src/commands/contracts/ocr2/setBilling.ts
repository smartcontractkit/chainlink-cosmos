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
      recommended_gas_price_uluna: input.recommendedGasPriceUluna,
    },
  }
}

const validateInput = (input: CommandInput): boolean => {
  let observationPayment: BN
  let transmissionPayment: BN

  const gasPrice: number = Number(input.recommendedGasPriceUluna) // parse as float64
  if (!isFinite(gasPrice)) {
    throw new Error(`recommendedGasPriceUluna=${input.recommendedGasPriceUluna} is not a valid floating point number.`)
  }

  if (gasPrice < 0.0) {
    throw new Error(`recommendedGasPriceUluna=${input.recommendedGasPriceUluna} cannot be negative`)
  }

  try {
    observationPayment = BN(input.observationPaymentGjuels)
    transmissionPayment = BN(input.transmissionPaymentGjuels) // parse as integers
  } catch {
    throw new Error(
      `observationPaymentGjuels=${input.observationPaymentGjuels} and ` +
        `transmissionPaymentGjuels=${input.transmissionPaymentGjuels} must both be integers`,
    )
  }
  if (observationPayment.isNeg() || transmissionPayment.isNeg()) {
    throw new Error(
      `observationPaymentGjuels=${input.observationPaymentGjuels} and ` +
        `transmissionPaymentGjuels=${input.transmissionPaymentGjuels} cannot be negative`,
    )
  }
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
