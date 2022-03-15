import { prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../lib/constants'

export default class MigrateContract extends TerraCommand {
  static description = '(NOT TESTED!) Upgrades a contract code id'
  static examples = [
    `yarn gauntlet tooling:migrate_contract --network=bombay-testnet --newCodeId=<CODE_ID_NUMBER> <CONTRACT_ADDRESS>`,
    `yarn gauntlet tooling:migrate_contract --network=bombay-testnet --newCodeId=2012 terra167ccv2h0z7k0p8j6qpuzwsgu5au5qvfwgmkjsl`,
  ]

  static id = 'tooling:migrate_contract'
  static category = CATEGORIES.TOOLING

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  makeRawTransaction = async () => {
    throw new Error('Upload command: makeRawTransaction method not implemented')
  }

  execute = async () => {
    const contractAddress = this.args[0]
    const codeId = this.flags.newCodeId

    await prompt(`Upgrading contract ${contractAddress} to a new code id ${codeId}, do you wish to continue?`)

    const tx = await this.migrateContract(contractAddress, codeId, {})

    return {
      responses: [
        {
          tx,
          contract: '',
        },
      ],
    }
  }
}
