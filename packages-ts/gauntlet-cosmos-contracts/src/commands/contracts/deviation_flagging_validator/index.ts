import Deploy from './deploy'
import { makeTransferOwnershipCommand, makeAcceptOwnershipCommand } from '../ownership'
import { CONTRACT_LIST } from '../../../lib/contracts'

export default [
  Deploy,
  makeTransferOwnershipCommand(CONTRACT_LIST.DEVIATION_FLAGGING_VALIDATOR),
  makeAcceptOwnershipCommand(CONTRACT_LIST.DEVIATION_FLAGGING_VALIDATOR),
]
