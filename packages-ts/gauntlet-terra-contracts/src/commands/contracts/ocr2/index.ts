import SetupFlow from './setup.dev.flow'
import OCR2InitializeFlow from './initialize.flow'
import Deploy from './deploy'
import SetBilling from './setBilling'
import ProposeConfig from './proposeConfig'
import ProposeOffchainConfig from './proposeOffchainConfig'
import Inspect from './inspection/inspect'
import ProposalCommands from './proposal'

export default [
  SetupFlow,
  Deploy,
  SetBilling,
  ProposeConfig,
  ProposeOffchainConfig,
  OCR2InitializeFlow,
  Inspect,
  ...ProposalCommands,
]
