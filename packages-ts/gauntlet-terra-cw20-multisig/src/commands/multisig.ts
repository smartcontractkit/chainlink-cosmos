import { Result } from '@chainlink/gauntlet-core'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { MsgExecuteContract } from '@terra-money/terra.js'

export const wrapCommand = (command) => {
  return class Multisig extends TerraCommand {
    command: TerraCommand

    static id = `${command.id}:multisig`

    constructor(flags, args) {
      super(flags, args)

      this.command = new command(flags, args)
    }

    makeRawTransaction = async () => {
      // TODO: Replace with Mulstig tx message
      return {} as MsgExecuteContract
    }

    execute = async () => {
      // If ID Proposal is provided, check the proposal status, and either approve or execute.
      // If ID Proposal is not provided, create a new proposal
      const message = await this.command.makeRawTransaction()
      logger.log('Command data:', message)

      return {} as Result<TransactionResponse>
    }
  }
}
