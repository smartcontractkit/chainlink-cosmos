import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { Result } from '@chainlink/gauntlet-core'
import { BN, logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { CATEGORIES } from '../../../lib/constants'

export default class TransferLink extends TerraCommand {
  static description = 'Transfer LINK'
  static examples = [
    `yarn gauntlet token:transfer --network=bombay-testnet --to=[RECEIVER] --amount=[AMOUNT_IN_TOKEN_UNITS]`,
    `yarn gauntlet token:transfer --network=bombay-testnet --to=[RECEIVER] --amount=[AMOUNT_IN_TOKEN_UNITS] --link=[TOKEN_ADDRESS] --decimals=[TOKEN_DECIMALS]`,
  ]

  static id = 'token:transfer'
  static category = CATEGORIES.LINK

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  execute = async () => {
    const decimals = this.flags.decimals || 18
    const link = this.flags.link || process.env.LINK
    const amount = new BN(this.flags.amount).mul(new BN(10).pow(new BN(decimals)))

    await prompt(`Sending ${this.flags.amount} LINK (${amount.toString()}) to ${this.flags.to}. Continue?`)
    const tx = await this.call(link, {
      transfer: {
        recipient: this.flags.to,
        amount: amount.toString(),
      },
    })
    logger.success(`LINK transferred successfully to ${this.flags.to} (txhash: ${tx.hash})`)
    return {
      responses: [
        {
          tx,
          contract: link,
        },
      ],
    } as Result<TransactionResponse>
  }
}
