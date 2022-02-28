import { BN, logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { defaultAfterExecute } from '../abstract/executionWrapper'
import { AccAddress, MsgSend } from '@terra-money/terra.js'
import { CATEGORIES, ULUNA_DECIMALS } from '../../lib/constants'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { Result } from '@chainlink/gauntlet-core'

type CommandInput = {
  destination: string
  // Units in LUNA
  amount: string
}

export default class TransferLuna extends TerraCommand {
  static description = 'Transfer Luna'
  static examples = [`yarn gauntlet wallet:transfer --network=bombay-testnet`]

  static id = 'wallet:transfer'
  static category = CATEGORIES.WALLET

  input: CommandInput

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  beforeExecute = async (input: CommandInput) => {
    await prompt(`Continue sending ${input.amount} uLUNA to ${input.destination}?`)
  }
  afterExecute = defaultAfterExecute

  makeInput = () => {
    return {
      destination: this.flags.to,
      amount: new BN(this.flags.amount).mul(new BN(10).pow(new BN(ULUNA_DECIMALS))),
    }
  }

  makeRawTransaction = async (signer: AccAddress) => {
    this.input = this.makeInput()
    if (!AccAddress.validate(this.input.destination)) throw new Error('Invalid destination address')
    return new MsgSend(signer, this.input.destination, `${this.input.amount.toString()}uluna`)
  }

  execute = async () => {
    const message = await this.makeRawTransaction(this.wallet.key.accAddress)
    await this.beforeExecute(this.input)
    const tx = await this.signAndSend([message])
    const result = {
      responses: [
        {
          tx,
          contract: '',
        },
      ],
    } as Result<TransactionResponse>
    await this.afterExecute(result)
    return result
  }
}
