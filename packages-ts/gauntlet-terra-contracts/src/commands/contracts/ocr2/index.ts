import SetupFlow from './setup.dev.flow'
import OCR2InitializeFlow from './initialize.flow'
import Deploy from './deploy'
import SetBilling from './setBilling'
import ProposeConfig from './proposeConfig'
import ProposeOffchainConfig from './proposeOffchainConfig'
import Inspection from './inspection'
import WithdrawPayment from './withdrawPayment'
import ProposalCommands from './proposal'
import CloseCommands from './close'
import { makeTransferOwnershipCommand, makeAcceptOwnershipCommand } from '../ownership'
import { CONTRACT_LIST } from '../../../lib/contracts'
import WithdrawFunds from './withdrawFunds'

export default [
  SetupFlow,
  Deploy,
  SetBilling,
  ProposeConfig,
  ProposeOffchainConfig,
  OCR2InitializeFlow,
  WithdrawPayment,
  WithdrawFunds,
  ...ProposalCommands,
  ...CloseCommands,
  ...Inspection,
  makeTransferOwnershipCommand(CONTRACT_LIST.OCR_2),
  makeAcceptOwnershipCommand(CONTRACT_LIST.OCR_2),
]
