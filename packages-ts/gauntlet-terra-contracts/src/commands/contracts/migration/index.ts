import { Result } from '@chainlink/gauntlet-core'
import { prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { AccAddress, MsgMigrateContract } from '@terra-money/terra.js'
import { CONTRACT_LIST } from '../../../lib/contracts'

type CommandInput<MigrationMessage> = {
  contract: string
  newCodeId: number
  migrationMsg: MigrationMessage
}

export type MigrationMessageMaker<MigrationMessage> = (flags, args) => MigrationMessage

export const makeMigrationCommand = <MigrationMessage>(
  contract: CONTRACT_LIST,
  makeMigrationMessage: MigrationMessageMaker<MigrationMessage>,
) => {
  return class MigrateContract extends TerraCommand {
    static description = `(NOT TESTED!) Upgrades a ${contract} contract instance to use a new Code ID. This new Code ID must expose a "migrate" function to migrate the contract state`
    static examples = [
      `yarn gauntlet ${contract}:migrate_contract --network=bombay-testnet --newCodeId=<CODE_ID_NUMBER> <CONTRACT_ADDRESS>`,
      `yarn gauntlet ${contract}:migrate_contract --network=bombay-testnet --newCodeId=2012 terra167ccv2h0z7k0p8j6qpuzwsgu5au5qvfwgmkjsl`,
    ]

    static id = `${contract}:migrate_contract`
    static category = contract

    input: CommandInput<MigrationMessage>

    constructor(flags, args: string[]) {
      super(flags, args)
    }

    buildCommand = async (flags, args): Promise<TerraCommand> => {
      this.input = this.makeInput(flags, args)
      this.validateInput(this.input)
      return this
    }

    beforeExecute = async () => {
      await prompt(`Continue upgrading contract ${this.input.contract} to new Code ID ${this.input.newCodeId}?`)
    }

    makeInput = (flags, args): CommandInput<MigrationMessage> => {
      return {
        newCodeId: Number(flags.newCodeId),
        contract: args[0],
        migrationMsg: makeMigrationMessage(flags, args),
      }
    }

    validateInput = (input: CommandInput<MigrationMessage>): boolean => {
      if (isNaN(input.newCodeId)) throw new Error(`Invalid Code ID: ${input.newCodeId}`)
      if (!AccAddress.validate(input.contract)) throw new Error('Invalid contract address')
      return true
    }

    makeRawTransaction = async (signer: AccAddress) => {
      return new MsgMigrateContract(signer, this.input.contract, this.input.newCodeId, this.input.migrationMsg as any)
    }

    execute = async () => {
      await this.buildCommand(this.flags, this.args)

      const message = await this.makeRawTransaction(this.wallet.key.accAddress)
      await this.beforeExecute()
      await prompt(
        `Upgrading contract ${this.input.contract} to a new code id ${this.input.newCodeId}, do you wish to continue?`,
      )
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
}
