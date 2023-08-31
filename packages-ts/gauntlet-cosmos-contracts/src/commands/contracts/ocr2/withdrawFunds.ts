import { BN, logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import {
  AbstractInstruction,
  BeforeExecute,
  ExecutionContext,
  instructionToCommand,
} from '../../abstract/executionWrapper'
import { CATEGORIES } from '../../../lib/constants'
import { AccAddress } from '@chainlink/gauntlet-cosmos'

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
  const queryAvailableLink = async () =>
    ((await context.provider.queryContractSmart(context.contract, { link_available_for_payment: {} } as any)) as any)
      .amount
  const amount = input.all ? await queryAvailableLink() : input.amount
  const recipient = input.recipient || context.signer.address
  return {
    amount: new BN(amount).toString(),
    recipient,
  }
}

const validateRecipient = async (input: CommandInput) => {
  if (input.recipient && AccAddress.validate(input.recipient))
    throw new Error(`Invalid recipient address ${input.recipient}`)
  return true
}

const validateRequiredAmount = async (input: CommandInput) => {
  if (!input.all && !input.amount) throw new Error(`An amount is required`)
  return true
}

const validateAmount = async (input: CommandInput) => {
  if (input.amount && isNaN(Number(input.amount))) throw new Error(`Invalid input amount ${input.amount}`)
  return true
}

// TODO: Deprecate
const validateInput = (input: CommandInput): boolean => true

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context, input) => async (signer) => {
  logger.info(`Withdrawing feed:
    - Amount: ${input.contract.amount}
    - Recipient: ${input.contract.recipient}
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
  validations: {
    validRecipient: validateRecipient,
    validAmount: validateAmount,
    requireAmount: validateRequiredAmount,
  },
}

export default instructionToCommand(withdrawFundsInstruction)
