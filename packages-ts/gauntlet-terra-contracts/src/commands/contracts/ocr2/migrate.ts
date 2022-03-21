import { CONTRACT_LIST } from '../../../lib/contracts'
import { MigrateContract, MigrationMessageMaker } from '../migration'

// TODO: Pending of migrate function on the contract. Serve this as an example of implementation
type OCR2MigrateMessage = any

const makeMigrationMessage: MigrationMessageMaker<OCR2MigrateMessage> = (flags, args) => {
  return {}
}

// export default makeMigrationCommand(CONTRACT_LIST.OCR_2, makeMigrateMessage)

export default class OCRMigrateCommand extends MigrateContract<OCR2MigrateMessage> {
  static description = MigrateContract.makeDescription(CONTRACT_LIST.OCR_2)
  static examples = MigrateContract.makeExamples(CONTRACT_LIST.OCR_2)

  static id = MigrateContract.makeId(CONTRACT_LIST.OCR_2)
  static category = MigrateContract.makeCategory(CONTRACT_LIST.OCR_2)

  makeMigrationMessage = makeMigrationMessage
}
