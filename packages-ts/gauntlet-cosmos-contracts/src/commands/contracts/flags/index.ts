import Deploy from './deploy'
import { makeTransferOwnershipCommand, makeAcceptOwnershipCommand } from '../ownership'
import { CONTRACT_LIST } from '../../../lib/contracts'

export default [
  Deploy,
  makeTransferOwnershipCommand(CONTRACT_LIST.FLAGS),
  makeAcceptOwnershipCommand(CONTRACT_LIST.FLAGS),
]
