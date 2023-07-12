import { makeAbstractCommand } from '.'
import { Result } from '@chainlink/gauntlet-core'
import { Client, CosmosCommand, TransactionResponse } from '@chainlink/gauntlet-cosmos'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { CATEGORIES } from '../../lib/constants'
import { CONTRACT_LIST } from '../../lib/contracts'
import { withAddressBook } from '../../lib/middlewares'

/**
 * Inspection commands need to match this interface
 * command: {
 *   contract: Contract related to the inspection
 *   id: Name of the command the user will execute
 * }
 * instructions: instruction[] Set of abstract query commands the inspection command will run
 * makeInput: Receives flags and args. Should return the input the underneath commands
 * makeInspectionInput: Transforms input into a comparable format
 * makeOnchainData: Parses every instruction command result to match the same interface the Inspection command expects
 * inspect: Compares both expected and onchain data.
 */
export interface InspectInstruction<CommandInput, ContractExpectedInfo> {
  command: {
    category: CATEGORIES
    contract: CONTRACT_LIST
    id: string
    examples: string[]
  }
  instructions: {
    contract: string
    function: string
  }[]
  makeInput: (flags: any, args: string[]) => Promise<CommandInput>
  makeOnchainData: (
    provider: Client,
  ) => (instructionsData: any[], input: CommandInput, contractAddress: string) => Promise<ContractExpectedInfo>
  inspect: (expected: CommandInput, data: ContractExpectedInfo) => boolean
}

export const instructionToInspectCommand = <CommandInput, Expected>(
  inspectInstruction: InspectInstruction<CommandInput, Expected>,
) => {
  const id = `${inspectInstruction.command.contract}:${inspectInstruction.command.id}`
  return class Command extends CosmosCommand {
    static id = id
    static examples = inspectInstruction.command.examples

    constructor(flags, args) {
      super(flags, args)
      this.use(withAddressBook)
    }

    makeRawTransaction = () => {
      throw new Error('Inspection command does not involve any transaction')
    }

    execute = async (): Promise<Result<TransactionResponse>> => {
      const input = await inspectInstruction.makeInput(this.flags, this.args)
      const commands = await Promise.all(
        inspectInstruction.instructions.map((instruction) =>
          makeAbstractCommand(`${instruction.contract}:${instruction.function}`, this.flags, this.args, input),
        ),
      )

      logger.loading('Fetching contract information...')
      const data = await Promise.all(
        commands.map(async (command) => {
          await command.invokeMiddlewares(command, command.middlewares)
          const { data } = await command.execute()
          return data
        }),
      )

      const onchainData = await inspectInstruction.makeOnchainData(this.provider)(data, input, this.args[0])
      const inspection = inspectInstruction.inspect(input, onchainData)
      return {
        data: inspection,
        responses: [
          {
            tx: {
              hash: '',
              wait: () => ({ success: inspection }),
            },
            contract: this.args[0],
          },
        ],
      }
    }
  }
}
