import { Result } from '@chainlink/gauntlet-core'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { AccAddress, MsgUpdateContractAdmin } from '@terra-money/terra.js'
import { CATEGORIES } from '../../lib/constants'

type CommandInput = {
  newAdmin: string
  contract: string
  force?: boolean
}

export default class UpdateContractAdmin extends TerraCommand {
  input: CommandInput

  static description = 'Updates contract admin. Admin role can migrate the contract to a new Code ID'
  static examples = [
    `yarn gauntlet tooling:update_contract_admin --network=bombay-testnet --to=<NEW_ADMIN> <CONTRACT_ADDRESS>`,
  ]

  static id = 'tooling:update_contract_admin'
  static category = CATEGORIES.TOOLING

  constructor(flags, args: string[]) {
    super(flags, args)
  }

  buildCommand = async (flags, args): Promise<TerraCommand> => {
    this.input = this.makeInput(flags, args)
    this.validateInput(this.input)
    return this
  }

  beforeExecute = async () => {
    await prompt(`Continue transferring contract ${this.input.contract} to new admin ${this.input.newAdmin}?`)
  }

  makeInput = (flags, args): CommandInput => {
    return {
      newAdmin: flags.to,
      contract: args[0],
      force: !!flags.force,
    }
  }

  validateInput = (input: CommandInput): boolean => {
    if (!AccAddress.validate(this.input.newAdmin)) throw new Error('Invalid new admin address')
    if (!AccAddress.validate(this.input.contract)) throw new Error('Invalid contract address')
    const knownMultisig = input.newAdmin === process.env.CW3_FLEX_MULTISIG
    if (input.force) {
      if (!knownMultisig) logger.warn(`New admin "${input.newAdmin}" might not be a multisig wallet`)
      else logger.success(`Proposed new admin is a known multisig wallet`)

      return true
    }
    // TODO: Update when Contracts expose its addresses
    if (!knownMultisig) throw new Error(`Proposed New admin "${input.newAdmin}"" is not a known multisig wallet`)
    else logger.success(`Proposed new admin is a known multisig wallet`)

    return true
  }

  makeRawTransaction = async (signer: AccAddress) => {
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
