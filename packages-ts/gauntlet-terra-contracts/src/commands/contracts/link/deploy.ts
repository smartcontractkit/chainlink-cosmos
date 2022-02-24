import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { Result } from '@chainlink/gauntlet-core'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { CATEGORIES, CW20_BASE_CODE_IDs } from '../../../lib/constants'
import { getRDD } from '../../../lib/rdd'

export default class DeployLink extends TerraCommand {
  static description = 'Deploys LINK token contract'
  static examples = [`yarn gauntlet token:deploy --network=bombay-testnet`]

  static id = 'token:deploy'
  static category = CATEGORIES.LINK

  static flags = {
    codeIDs: { description: 'The path to contract code IDs file' },
    artifacts: { description: 'The path to contract artifacts folder' },
  }

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  makeRawTransaction = async () => {
    throw new Error('Deploy LINK command: makeRawTransaction method not implemented')
  }

  execute = async () => {
    let cw20_base = CW20_BASE_CODE_IDs[this.flags.network]
    if (!cw20_base) {
      console.log(`The hardcoded codeId for the network "${this.flags.network}" does not exist.`)
      const codeIdData = getRDD(this.flags.codeIDs, 'CodeIds')
      if (codeIdData.cw20_base) {
        await prompt(
          `The hardcoded codeId for the network "${this.flags.network}" does not exist, do you wish to proceed using the codeId found in the associated codeIDs file of "${codeIdData.cw20_base}"?`,
        )
        cw20_base = codeIdData.cw20_base
      } else {
        throw new Error(
          `No codeid was found in the hardcoded values or the codeIDs file associated with this this network: "${this.flags.network}"`,
        )
      }
    }

    await prompt(`Begin deploying LINK Token?`)

    const deploy = await this.deploy(cw20_base, {
      name: 'ChainLink Token',
      symbol: 'LINK',
      decimals: 18,
      initial_balances: [{ address: this.wallet.key.accAddress, amount: '1000000000000000000000000000' }],
      marketing: {
        project: 'Chainlink',
        logo: {
          url:
            'https://assets-global.website-files.com/5e8c4efdc725c62673645017/5e981c33430c9765dba5a098_Symbol%20White.svg',
        },
      },
      mint: {
        minter: this.wallet.key.accAddress,
      },
    })
    const result = await this.provider.wasm.contractQuery(deploy.address!, { token_info: {} })
    logger.success(`LINK token successfully deployed at ${deploy.address} (txhash: ${deploy.hash})`)
    logger.debug(result)
    return {
      responses: [
        {
          tx: deploy,
          contract: deploy.address,
        },
      ],
    } as Result<TransactionResponse>
  }
}
