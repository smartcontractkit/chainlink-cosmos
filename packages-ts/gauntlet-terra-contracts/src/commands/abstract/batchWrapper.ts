import AbstractCommand, { makeAbstractCommand } from '.'
import { Result, WriteCommand } from '@chainlink/gauntlet-core'
import { TerraCommand, TransactionResponse, logger } from '@chainlink/gauntlet-terra'
import { AccAddress, LCDClient, Msg } from '@terra-money/terra.js'
import { prompt } from '@chainlink/gauntlet-core/dist/utils'
import { ExecutionContext } from './executionWrapper'

export const wrapCommand = (command) => {
  return class BatchCommand extends TerraCommand {
    static id = `${command.id}:batch`
    command: AbstractCommand
    subCommands: TerraCommand[]

    constructor(flags, args) {
      super(flags, args)
    }

    buildCommand = async (flags, args): Promise<TerraCommand> => {
      //   console.log(command)
      //   console.log(Object.getPrototypeOf(command) instanceof TerraCommand)
      //   console.log(TerraCommand.prototype)
      //   console.log(command instanceof TerraCommand)
      //   console.log(flags)
      //   console.log(command.buildCommand)
      // const abstractCommand = await makeAbstractCommand(BatchCommand.id, flags, args)
      // await abstractCommand.invokeMiddlewares(abstractCommand, abstractCommand.middlewares)
      // this.command = abstractCommand
      this.subCommands = flags.input.map(async (item) => {
        return command.buildCommand ? await command.buildCommand(item, args) : command
      })
      return this
    }

    makeRawTransaction = async (signer: AccAddress) => {
      return this.subCommands[0].makeRawTransaction(signer)
      // return this.subCommands.map((element) => element.makeRawTransaction(signer))
    }

    defaultBeforeBatchExecute = () => async () => {
      logger.loading(`Executing ${command.id} from contract ${this.args[0]} for the following sets of inputs`)
      this.subCommands.forEach((element) => logger.log('Input Params:'))
      await prompt(`Continue?`)
    }

    execute = async (): Promise<Result<TransactionResponse>> => {
      if (typeof (command as TerraCommand).buildCommand == 'undefined')
        throw new Error('This command does not support batching')
      // TODO: Command should be built from gauntet-core
      await this.buildCommand(this.flags, this.args)
      await this.defaultBeforeBatchExecute

      // console.log(this.subCommands[0].wallet)
      const msg = await this.subCommands[0].makeRawTransaction(this.subCommands[0].wallet.key.accAddress)
      console.log(msg)
      let tx = await this.subCommands[0].signAndSend([msg])
      const response = {
        responses: [
          {
            tx,
            contract: this.args[0],
          },
        ],
      } as Result<TransactionResponse>
      const data = await this.command.afterExecute(response)
      return !!data ? { ...response, data: { ...data } } : response
    }
  }
}
