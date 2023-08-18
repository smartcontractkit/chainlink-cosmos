import AddAccess from './addAccess'
import RemoveAccess from './removeAccess'
import Deploy from './deploy'
import { makeTransferOwnershipCommand, makeAcceptOwnershipCommand } from '../ownership'
import { CONTRACT_LIST } from '../../../lib/contracts'

export default [
  Deploy,
  AddAccess,
  RemoveAccess,
  makeTransferOwnershipCommand(CONTRACT_LIST.ACCESS_CONTROLLER),
  makeAcceptOwnershipCommand(CONTRACT_LIST.ACCESS_CONTROLLER),
]
