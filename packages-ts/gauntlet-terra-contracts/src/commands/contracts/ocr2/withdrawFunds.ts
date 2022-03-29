import { BN, logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import {
  AbstractInstruction,
  BeforeExecute,
  ExecutionContext,
  instructionToCommand,
} from '../../abstract/executionWrapper'
import { CATEGORIES } from '../../../lib/constants'
import { AccAddress } from '@terra-money/terra.js'

type CommandInput = {
  amount?: string
  recipient?: string
  all?: boolean
}

type ContractInput = {
  amount: string
  recipient: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {
    amount: flags.amount,
    recipient: flags.recipient,
    all: flags.all,
  }
}

const makeContractInput = async (input: CommandInput, context: ExecutionContext): Promise<ContractInput> => {
  const amount = input.all
    ? ((await context.provider.wasm.contractQuery(context.contract, 'link_available_for_payment' as any)) as any).amount
    : input.amount
  const recipient = input.recipient || context.wallet.key.accAddress
  return {
    amount: new BN(amount).toString(),
    recipient,
  }
}

const validateInput = (input: CommandInput): boolean => {
  if (input.recipient && AccAddress.validate(input.recipient))
    throw new Error(`Invalid recipient address ${input.recipient}`)
  if (!input.all && !input.amount) throw new Error(`An amount is required`)
  if (input.amount && isNaN(Number(input.amount))) throw new Error(`Invalid input amount ${input.amount}`)
  return true
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context, inputContext) => async (signer) => {
  logger.info(`Withdrawing feed:
    - Amount: ${inputContext.contractInput.amount}
    - Recipient: ${inputContext.contractInput.recipient}
  `)

  await prompt('Continue?')
  return
}

const withdrawFundsInstruction: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.OCR,
    contract: 'ocr2',
    function: 'withdraw_funds',
  },
  examples: [
    'yarn gauntlet ocr2:withdraw_funds --network=<NETWORK> --all --recipient=<RECIPIENT_ADDRESS> <CONTRACT_ADDRESS>',
    'yarn gauntlet ocr2:withdraw_funds --network=<NETWORK> --amount=<AMOUNT> <CONTRACT_ADDRESS>',
  ],
  makeInput: makeCommandInput,
  validateInput,
  makeContractInput,
  beforeExecute,
}

export default instructionToCommand(withdrawFundsInstruction)
