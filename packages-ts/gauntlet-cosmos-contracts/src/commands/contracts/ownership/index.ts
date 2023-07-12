import { makeTransferOwnershipInstruction } from './transferOwnership'
import { makeAcceptOwnershipInstruction } from './acceptOwnership'
import { instructionToCommand } from '../../abstract/executionWrapper'
import { CONTRACT_LIST } from '../../../lib/contracts'

export const makeTransferOwnershipCommand = (contractId: CONTRACT_LIST) => {
  return instructionToCommand(makeTransferOwnershipInstruction(contractId))
}

export const makeAcceptOwnershipCommand = (contractId: CONTRACT_LIST) => {
  return instructionToCommand(makeAcceptOwnershipInstruction(contractId))
}
