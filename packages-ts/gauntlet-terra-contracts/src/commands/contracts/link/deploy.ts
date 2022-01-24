import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { Result } from '@chainlink/gauntlet-core'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { CATEGORIES, CW20_BASE_CODE_IDs } from '../../../lib/constants'

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

  execute = async () => {
    await prompt(`Begin deploying LINK Token?`)
    const deploy = await this.deploy(CW20_BASE_CODE_IDs[this.flags.network], {
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
