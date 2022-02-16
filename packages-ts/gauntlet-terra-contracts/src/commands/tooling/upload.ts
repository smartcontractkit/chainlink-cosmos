import { logger, io, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand } from '@chainlink/gauntlet-terra'
import { CONTRACT_LIST, getContract } from '../../lib/contracts'
import { CATEGORIES } from '../../lib/constants'
import path from 'path'
export default class UploadContractCode extends TerraCommand {
  static description = 'Upload cosmwasm contract artifacts'
  static examples = [
    `yarn gauntlet upload --network=bombay-testnet`,
    `yarn gauntlet upload --network=bombay-testnet [contract names]`,
    `yarn gauntlet upload --network=bombay-testnet flags cw20_base`,
  ]

  static id = 'upload'
  static category = CATEGORIES.TOOLING

  static flags = {
    version: { description: 'The version to retrieve artifacts from (Defaults to v0.0.4)' },
    maxRetry: { description: 'The number of times to retry failed uploads (Defaults to 5)' },
  }

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  getCodeId(response): number | undefined {
    return Number(this.parseResponseValue(response, 'store_code', 'code_id'))
  }

  execute = async () => {
    const askedContracts = !!this.args.length
      ? Object.keys(CONTRACT_LIST)
          .filter((contractId) => this.args.includes(CONTRACT_LIST[contractId]))
          .map((contractId) => CONTRACT_LIST[contractId])
      : Object.values(CONTRACT_LIST)

    const contractsToOverride = askedContracts.filter((contractId) => Object.keys(this.codeIds).includes(contractId))
    if (contractsToOverride.length > 0) {
      logger.info(`The following contracts are deployed already and will be overwritten: ${contractsToOverride}`)
    }

    await prompt(`Continue uploading the following contract codes: ${askedContracts}?`)

    const contractReceipts = {}
    const responses: any[] = []
    const parsedRetryCount = parseInt(this.flags.maxRetry)
    const maxRetry = parsedRetryCount ? parsedRetryCount : 5
    for (let contractName of askedContracts) {
      await prompt(`Uploading contract ${contractName}, do you wish to continue?`)
      const contract = await getContract(contractName, this.flags.version)
      console.log('CONTRACT Bytecode exists:', !!contract.bytecode)
      for (let retry = 0; retry < maxRetry; retry++) {
        try {
          const res = await this.upload(contract.bytecode, contractName)

          logger.success(`Contract ${contractName} code uploaded succesfully`)
          contractReceipts[contractName] = res.tx
          responses.push({
            tx: res,
            contract: null,
          })
        } catch (e) {
          const message = e.response.data.message || e.message
          logger.error(`Error deploying ${contractName} on attempt ${retry + 1} with the error: ${message}`)
          if (maxRetry === retry + 1) {
            throw new Error(message)
          }
          continue
        }
        break
      }
    }

    const codeIds = Object.keys(contractReceipts).reduce(
      (agg, contractName) => ({
        ...agg,
        [contractName]: this.getCodeId(contractReceipts[contractName]) || this.codeIds[contractName] || '',
      }),
      this.codeIds,
    )

    io.saveJSON(
      codeIds,
      path.join(process.cwd(), `./packages-ts/gauntlet-terra-contracts/codeIds/${this.flags.network}`),
    )
    logger.success('New code ids have been saved')

    return {
      responses,
    }
  }
}
