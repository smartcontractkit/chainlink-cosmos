import Deploy from './deploy'
import ProposeContract from './proposeContract'
import ConfirmContract from './confirmContract'
import { makeTransferOwnershipCommand, makeAcceptOwnershipCommand } from '../ownership'
import { CONTRACT_LIST } from '../../../lib/contracts'

export default [
  Deploy,
  ProposeContract,
  ConfirmContract,
  makeTransferOwnershipCommand(CONTRACT_LIST.PROXY_OCR_2),
  makeAcceptOwnershipCommand(CONTRACT_LIST.PROXY_OCR_2),
]
