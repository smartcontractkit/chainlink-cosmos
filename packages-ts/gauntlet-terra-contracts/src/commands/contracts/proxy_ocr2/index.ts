import Deploy from './deploy'
import ProposeContract from './proposeContract'
import ConfirmContract from './confirmContract'
import { makeTransferOwnershipCommand, makeAcceptOwnershipCommand } from '../ownership'
import { CONTRACT_LIST } from '../../../lib/contracts'
import Inspect from './inspection/inspect'

export default [
  Deploy,
  ProposeContract,
  ConfirmContract,
  Inspect,
  makeTransferOwnershipCommand(CONTRACT_LIST.PROXY_OCR_2),
  makeAcceptOwnershipCommand(CONTRACT_LIST.PROXY_OCR_2),
]
