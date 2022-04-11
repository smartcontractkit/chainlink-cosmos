import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { Result } from '@chainlink/gauntlet-core'
import { AbstractInstruction, instructionToCommand, BeforeExecute } from '../../abstract/executionWrapper'
import { TransactionResponse } from '@chainlink/gauntlet-terra'
import { AccAddress } from '@terra-money/terra.js'
import { CATEGORIES } from '../../../lib/constants'
import { CONTRACT_LIST } from '../../../lib/contracts'
import { parseOraclePaidEvent } from '../../../lib/events'

type CommandInput = {
  transmitter: string
}

type ContractInput = {
  transmitter: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  return {
    transmitter: flags.transmitter,
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    transmitter: input.transmitter,
  }
}

const validateInput = (input: CommandInput): boolean => {
  if (!AccAddress.validate(input.transmitter)) throw new Error(`Invalid ocr2 contract address`)

  return true
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context) => async () => {
  logger.info(`Transmitter ${context.contractInput.transmitter} withdrawing LINK payment from ${context.contract}`)

  await prompt('Continue?')
  return
}

const afterExecute = () => async (response: Result<TransactionResponse>) => {
  const events = response.responses[0].tx.events
  if (!events) {
    logger.error('Could not retrieve events from tx')
    return
  }

  const paidOracleEvent = parseOraclePaidEvent(events[0].wasm)
  if (!paidOracleEvent) {
    logger.error('Unable to parse/validate response data')
    return
  }

  logger.info(`Paying ${paidOracleEvent.payee} ${paidOracleEvent.amount} LINK`)
  return
}

const withdrawPaymentInstruction: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.OCR,
    contract: 'ocr2',
    function: 'withdraw_payment',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  beforeExecute: beforeExecute,
  afterExecute: afterExecute,
}

export default instructionToCommand(withdrawPaymentInstruction)
