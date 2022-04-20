import { assertions, logger } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../lib/constants'

export default class QueryContractCode extends TerraCommand {
  static description = 'Query deployed contracts'
  static examples = [`yarn gauntlet tooling:query --network=[NETWORK] --msg='QUERY' [CONTRACT_ADDRESS]`]

  static id = 'tooling:query'
  static category = CATEGORIES.TOOLING

  static flags = {}

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  makeRawTransaction = async () => {
    throw new Error('Query command: makeRawTransaction method not implemented')
  }

  execute = async () => {
    // Assert that --msg exists
    assertions.assert(!!this.flags.msg, `Message required, please specify a --msg`)
    // Assert that only one argument was inputted
    assertions.assert(this.args.length == 1, `Expected 1 argument, got ${this.args.length}`)
    // Execute query
    const responses: any[] = []
    try {
      const result = await this.query(this.args[0], this.flags.msg)
      logger.info(`Query finished with result: ${JSON.stringify(result)}`)
      responses.push(result)
    } catch (error) {
      logger.error(`Failed to query contract: ${error}`)
    }
    // Return response
    return {
      responses,
    }
  }
}
