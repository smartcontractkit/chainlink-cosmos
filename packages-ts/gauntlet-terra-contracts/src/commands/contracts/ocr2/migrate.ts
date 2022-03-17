import { CONTRACT_LIST } from '../../../lib/contracts'
import { makeMigrationCommand, MigrationMessageMaker } from '../migration'

// TODO: Pending of migrate function on the contract. Serve this as an example of implementation
type OCR2MigrateMessage = any

const makeMigrateMessage: MigrationMessageMaker<OCR2MigrateMessage> = (flags, args) => {
  return {}
}

export default makeMigrationCommand(CONTRACT_LIST.OCR_2, makeMigrateMessage)
