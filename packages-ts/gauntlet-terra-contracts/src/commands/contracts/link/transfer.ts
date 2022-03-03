import { BN, logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { CATEGORIES, TOKEN_DECIMALS } from '../../../lib/constants'
import { AbstractInstruction, ExecutionContext, instructionToCommand } from '../../abstract/executionWrapper'
import { AccAddress } from '@terra-money/terra.js'

type CommandInput = {
  to: string
  // Units in LINK
  amount: string
}

type ContractInput = {
  recipient: string
  amount: string
}

const makeCommandInput = async (flags: any): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {
    to: flags.to,
    amount: flags.amount,
  }
}

const validateInput = (input: CommandInput): boolean => {
  if (!AccAddress.validate(input.to)) throw new Error(`Invalid destination address`)
  return true
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  const amount = new BN(input.amount).mul(new BN(10).pow(new BN(TOKEN_DECIMALS)))
  return {
    recipient: input.to,
    amount: amount.toString(),
  }
}

const beforeExecute = (context: ExecutionContext<CommandInput, ContractInput>) => async (): Promise<void> => {
  logger.info(
    `Transferring ${context.contractInput.amount} (${context.input.amount}) Tokens to ${context.contractInput.recipient}`,
  )
  await prompt('Continue?')
}

const transferToken: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.LINK,
    contract: 'cw20_base',
    function: 'transfer',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  beforeExecute,
}

export default instructionToCommand(transferToken)
