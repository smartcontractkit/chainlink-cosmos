import { Result } from '@chainlink/gauntlet-core'
import { CosmosCommand, TransactionResponse, logger } from '@chainlink/gauntlet-cosmos'
// import { MsgExecuteContract } from "cosmjs-types/cosmwasm/wasm/v1/tx";
import { AccAddress } from '@chainlink/gauntlet-cosmos'
import { prompt } from '@chainlink/gauntlet-core/dist/utils'
import { RDD } from '@chainlink/gauntlet-cosmos'
import { EncodeObject } from '@cosmjs/proto-signing'

export const wrapCommand = (command) => {
  return class BatchCommand extends CosmosCommand {
    static id = `${command.id}:batch`
    subCommands: CosmosCommand[]

    constructor(flags, args) {
      super(flags, args)
    }

    validateInput = (input, flags, args): Boolean => {
      if (!input) {
        if (!flags.rdd) throw new Error(`One of --input, --inputFile, or --rdd must be provided`)
        return true
      }

      const invalidInputConditions = args.length != input.length && args.length != 1
      if (invalidInputConditions)
        throw new Error(`Cannot apply ${input.length} command inputs to ${args.length} contracts`)
      return true
    }

    buildDefaultCommands = async (flags, args): Promise<CosmosCommand[]> => {
      return await Promise.all(
        args.map(async (contract, idx) => {
          let c = new command(flags, [contract]) as CosmosCommand
          await c.invokeMiddlewares(c, c.middlewares)
          c = c.buildCommand ? await c.buildCommand(flags, [contract]) : c
          return c
        }),
      )
    }

    buildCommandsFromInput = async (input, flags, args): Promise<CosmosCommand[]> => {
      return await Promise.all(
        input.map(async (individualInput, idx) => {
          const newFlags = { ...flags, ...individualInput }

          const individualArgs = args.length == 1 ? args : [args[idx]]
          let c = new command(newFlags, individualArgs) as CosmosCommand
          await c.invokeMiddlewares(c, c.middlewares)
          c = c.buildCommand ? await c.buildCommand(newFlags, individualArgs) : c
          return c
        }),
      )
    }

    buildCommand = async (flags, args): Promise<CosmosCommand> => {
      const input = flags.input
        ? flags.input
        : flags.inputFile
        ? RDD.parseJSON(flags.inputFile, 'BatchInputFile')
        : null

      this.validateInput(input, flags, args)

      this.subCommands = input
        ? await this.buildCommandsFromInput(input, flags, args)
        : await this.buildDefaultCommands(flags, args)

      this.afterExecute =
        this.subCommands[0].afterExecute.toString() === this.afterExecute.toString()
          ? this.afterExecute
          : this.afterExecuteOverride
      return this
    }

    beforeExecute = async (signer) => {
      logger.line()
      for (const command of this.subCommands) {
        await command.beforeExecute(signer)
        logger.line()
      }
    }

    afterExecuteOverride = async (response) => {
      logger.line()
      for (const command of this.subCommands) {
        await command.afterExecute(response)
        logger.line()
      }
    }

    simulateExecute_ = async (msgs: EncodeObject[]) => {
      const signer = this.signer.address // signer is the default loaded wallet
      logger.loading(`Executing batch ${command.id} tx simulation`)

      const estimatedGas = await this.simulate(signer, msgs)
      logger.info(`Tx simulation successful: estimated gas usage is ${estimatedGas}`)
      return estimatedGas
    }

    makeRawTransaction = async (signer: AccAddress) => {
      const rawTxs = (await Promise.all(this.subCommands.map((c) => c.makeRawTransaction(signer)))).reduce(
        (agg, txs) => [...agg, ...txs],
      )
      return rawTxs
    }

    execute = async (): Promise<Result<TransactionResponse>> => {
      await this.buildCommand(this.flags, this.args)

      const msgs = await this.makeRawTransaction(this.signer.address)
      await this.simulateExecute_(msgs)

      await this.beforeExecute(this.signer.address)
      await prompt(`Continue?`)

      const tx = await this.signAndSend(msgs)
      const response = {
        responses: [
          {
            tx,
            contract: this.args[0],
          },
        ],
      } as Result<TransactionResponse>
      const data = await this.afterExecute(response)
      return !!data ? { ...response, data: { ...data } } : response
    }
  }
}
