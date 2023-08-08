import { AccAddress } from '@chainlink/gauntlet-cosmos'
import { logger } from '@chainlink/gauntlet-cosmos'
import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { CATEGORIES, TOKEN_DECIMALS } from '../../../lib/constants'
import { AbstractInstruction, BeforeExecute, instructionToCommand } from '../../abstract/executionWrapper'

type CommandInput = {
  to: string
  // Units in LINK
  amount: string
}

type ContractInput = {
  recipient: string
  amount: string
}

const makeCommandInput = async (flags, args): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {
    to: flags.to,
    amount: flags.amount,
  }
}

const validateInput = (input) => {
  if (!AccAddress.validate(input.to)) throw new Error(`Invalid destination address`)
  if (isNaN(Number(input.amount))) throw new Error(`Amount ${input.amount} is not a number`)
  return true
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  const amount = new BN(input.amount).mul(new BN(10).pow(new BN(TOKEN_DECIMALS)))
  return {
    recipient: input.to,
    amount: amount.toString(),
  }
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (_, input) => async () => {
  logger.info(
    `Minting ${input.contract.amount} (${input.user.amount}) Tokens to ${logger.styleAddress(
      input.contract.recipient,
    )}`,
  )
}

const transferToken: AbstractInstruction<CommandInput, ContractInput> = {
  examples: [`yarn gauntlet cw20_base:mint --network=<NETWORK> --to=<ACCOUNT> --amount=<AMOUNT> <CONTRACT_ADDRESS>`],
  instruction: {
    category: CATEGORIES.LINK,
    contract: 'cw20_base',
    function: 'mint',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  beforeExecute,
}

export default instructionToCommand(transferToken)
