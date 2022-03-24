import { CONTRACT_LIST } from '../../../lib/contracts'
import { MigrateContract } from '../migration'

// Notice: Msg used to migrate OCR2 from v0.1.5 to v1.0.0
type MsgMigrate_1_0_0 = any

export default class MigrateCommand extends MigrateContract<MsgMigrate_1_0_0> {
  static description = MigrateContract.makeDescription(CONTRACT_LIST.OCR_2)
  static examples = MigrateContract.makeExamples(CONTRACT_LIST.OCR_2)

  static id = MigrateContract.makeId(CONTRACT_LIST.OCR_2)
  static category = MigrateContract.makeCategory(CONTRACT_LIST.OCR_2)

  makeMigrationMessage = (flags, args) => ({})
}
