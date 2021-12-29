import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { Result } from '@chainlink/gauntlet-core'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { CATEGORIES } from '../../../lib/constants'

export default class TransferLink extends TerraCommand {
  static description = 'Transfer LINK'
  static examples = [
    `yarn gauntlet transfer:token --network=bombay-testnet --to=AQoKYV7tYpTrFZN6P5oUufbQKAUr9mNYGe1TTJC9wajM --amount=100`,
  ]

  static id = 'transfer:token'
  static category = CATEGORIES.LINK

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  execute = async () => {
    const tx = await this.call(process.env.LINK, {
      transfer: {
        recipient: this.flags.to,
        amount: this.flags.amount,
      },
    })
    logger.success(`LINK transferred successfully to ${this.flags.to} (txhash: ${tx.hash})`)
    return {
      responses: [
        {
          tx,
          contract: process.env.LINK,
        },
      ],
    } as Result<TransactionResponse>
  }
}
