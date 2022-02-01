import { logger, io, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand } from '@chainlink/gauntlet-terra'
import { CONTRACT_LIST, getContract } from '../../lib/contracts'
import { CATEGORIES, DEFAULT_RELEASE_VERSION } from '../../lib/constants'
import path from 'path'
export default class UploadContractCode extends TerraCommand {
  static description = 'Upload cosmwasm contract artifacts'
  static examples = [
    `yarn gauntlet upload --network=bombay-testnet`,
    `yarn gauntlet upload --network=bombay-testnet [contract names]`,
    `yarn gauntlet upload --network=bombay-testnet flags link_token`,
  ]

  static id = 'upload'
  static category = CATEGORIES.TOOLING

  static flags = {
    version: { description: 'The version to retrieve artifacts from (Defaults to v0.0.4)' },
    codeIDs: { description: 'The path to contract code IDs file' },
    artifacts: { description: 'The path to contract artifacts folder' },
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
    for (const contractName of askedContracts) {
      const version = this.flags.version ? this.flags.version : DEFAULT_RELEASE_VERSION
      const contract = await getContract(contractName, version)
      console.log('CONTRACT Bytecode exists:', !!contract.bytecode)

      try {
        const res = await this.upload(contract.bytecode, contractName)

        logger.success(`Contract ${contractName} code uploaded succesfully`)
        contractReceipts[contractName] = res.tx
        responses.push({
          tx: res,
          contract: null,
        })
      } catch (e) {
        logger.error(`Error deploying ${contractName} code: ${e.message}`)
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
