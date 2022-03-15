import { Result } from '@chainlink/gauntlet-core'
import { prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { AccAddress, MsgUpdateContractAdmin } from '@terra-money/terra.js'
import { CATEGORIES } from '../../lib/constants'

type CommandInput = {
  newAdmin: string
  contract: string
}

export default class UpdateContractAdmin extends TerraCommand {
  input: CommandInput

  static description = 'Updates contract admin'
  static examples = [
    `yarn gauntlet tooling:transfer_contract_admin --network=bombay-testnet --to=<NEW_ADMIN> <CONTRACT_ADDRESS>`,
  ]

  static id = 'tooling:transfer_contract_admin'
  static category = CATEGORIES.TOOLING

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  buildCommand = async (flags, args): Promise<TerraCommand> => {
    this.input = this.makeInput(flags, args)
    return this
  }

  beforeExecute = async () => {
    await prompt(`Continue transferring contract ${this.input.contract} to new admin ${this.input.newAdmin}?`)
  }

  makeInput = (flags, _) => {
    return {
      newAdmin: flags.to,
      contract: this.args[0],
    }
  }

  makeRawTransaction = async (signer: AccAddress) => {
    if (!AccAddress.validate(this.input.newAdmin)) throw new Error('Invalid new admin address')
    if (!AccAddress.validate(this.input.contract)) throw new Error('Invalid contract address')
    return new MsgUpdateContractAdmin(signer, this.input.newAdmin, this.input.contract)
  }

  execute = async () => {
    await this.buildCommand(this.flags, this.args)

    const message = await this.makeRawTransaction(this.wallet.key.accAddress)
    await this.beforeExecute()
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
