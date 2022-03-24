import { Result } from '@chainlink/gauntlet-core'
import { prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { AccAddress, MsgMigrateContract } from '@terra-money/terra.js'

type CommandInput<MsgMigrate> = {
  contract: string
  newCodeId: number
  migrationMsg: MsgMigrate
}

export type MigrationMessageMaker<MsgMigrate> = (flags, args) => MsgMigrate

export abstract class MigrateContract<MsgMigrate> extends TerraCommand {
  static makeDescription = (contract) =>
    `(NOT TESTED!) Upgrades a contract ${contract} instance to use a new Code ID. This new Code ID must expose a "migrate" function to migrate the contract state`
  static makeExamples = (contract) => [
    `yarn gauntlet ${contract}:migrate_contract --network=<NETWORK> --newCodeId=<CODE_ID_NUMBER> <CONTRACT_ADDRESS>`,
    `yarn gauntlet ${contract}:migrate_contract --network=mainnet --newCodeId=2012 terra167ccv2h0z7k0p8j6qpuzwsgu5au5qvfwgmkjsl`,
  ]

  static makeId = (contract) => `${contract}:migrate_contract`
  static makeCategory = (contract) => contract

  abstract makeMigrationMessage: MigrationMessageMaker<MsgMigrate>
  input: CommandInput<MsgMigrate>

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

  makeInput = (flags, args): CommandInput<MsgMigrate> => {
    return {
      newCodeId: Number(flags.newCodeId),
      contract: args[0],
      migrationMsg: this.makeMigrationMessage(flags, args),
    }
  }

  validateInput = (input: CommandInput<MsgMigrate>): boolean => {
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
