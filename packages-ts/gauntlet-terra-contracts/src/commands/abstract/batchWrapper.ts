import AbstractCommand, { makeAbstractCommand } from '.'
import { Result, WriteCommand } from '@chainlink/gauntlet-core'
import { TerraCommand, TransactionResponse, logger } from '@chainlink/gauntlet-terra'
import { AccAddress, LCDClient, Msg } from '@terra-money/terra.js'
import { prompt } from '@chainlink/gauntlet-core/dist/utils'
import { ExecutionContext } from './executionWrapper'
import { parseJSON } from '@chainlink/gauntlet-terra/dist/lib/rdd'

export const wrapCommand = (command) => {
  return class BatchCommand extends TerraCommand {
    static id = `${command.id}:batch`
    subCommands: any[]

    constructor(flags, args) {
      super(flags, args)
    }

    buildCommand = async (flags, args): Promise<TerraCommand> => {
      const input = flags.input ? flags.input : parseJSON(flags.inputFile, 'BatchInputFile')

      this.subCommands = await Promise.all(
        input.map(async (individualInput) => {
          const newFlags = { ...flags, input: individualInput }

          let c = new command(newFlags, args) as TerraCommand
          await c.invokeMiddlewares(c, c.middlewares)
          c = c.buildCommand ? await c.buildCommand(newFlags, args) : c
          return c
        }),
      )
      return this
    }

    makeRawTransaction = async (signer: AccAddress) => {
      const rawTxs = (await Promise.all(this.subCommands.map((c) => 
      c.makeRawTransaction(signer)))).reduce((agg, txs) => [
        ...agg,
        ...txs,
      ])
      return rawTxs
    }

    execute = async (): Promise<Result<TransactionResponse>> => {
      // TODO: Command should be built from gauntet-core
      await this.buildCommand(this.flags, this.args)
      await Promise.all(this.subCommands.map(async (element) => await element.command.simulateExecute()))

      logger.loading(`Executing ${command.id} from contract ${this.args[0]} for the following sets of inputs`)
      let x = this.subCommands.map((element) => element.command.params)
      logger.log('Input Params:', x)
      await prompt(`Continue?`)

      const msgs = await this.makeRawTransaction(this.subCommands[0].wallet.key.accAddress)
      let tx = await this.subCommands[0].signAndSend(msgs)
      const response = {
        responses: [
          {
            tx,
            contract: this.args[0],
          },
        ],
      } as Result<TransactionResponse>
      const data = await this.subCommands[0].afterExecute(response)
      return !!data ? { ...response, data: { ...data } } : response
    }
  }
}
