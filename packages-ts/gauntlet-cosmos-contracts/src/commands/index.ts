import Upload from './tooling/upload'
import Query from './tooling/query'
import TransferLink from './contracts/link/transfer'
import MintLink from './contracts/link/mint'
import DeployLink from './contracts/link/deploy'
import OCR2 from './contracts/ocr2'
import AccessController from './contracts/access_controller'
import Flags from './contracts/flags'
import Proxy_OCR2 from './contracts/proxy_ocr2'
import DeviationFlaggingValidator from './contracts/deviation_flagging_validator'
import Multisig from './contracts/multisig'
import CW4_GROUP from './contracts/cw4_group'
import Wallet from './wallet'

export default [
  Upload,
  Query,
  DeployLink,
  TransferLink,
  MintLink,
  ...OCR2,
  ...AccessController,
  ...Flags,
  ...DeviationFlaggingValidator,
  ...Proxy_OCR2,
  ...Multisig,
  ...CW4_GROUP,
  ...Wallet,
]
