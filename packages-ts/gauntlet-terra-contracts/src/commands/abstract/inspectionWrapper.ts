import AbstractCommand, { makeAbstractCommand } from '.'
import { Result } from '@chainlink/gauntlet-core'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'

export type InspectionInput<CommandInput, Expected> = {
  commandInput?: CommandInput
  expected: Expected
}

/**
 * Inspection commands need to match this interface
 * command: {
 *   contract: Contract related to the inspection
 *   id: Name of the command the user will execute
 * }
 * instructions: instruction[] Set of abstract query commands the inspection command will run
 * makeInput: Receives flags and args. Should return the input the underneath commands, and the expected result we want
 * makeOnchainData: Parses every instruction command result to match the same interface the Inspection command expects
 * inspect: Compares both expected and onchain data.
 */
export interface InspectInstruction<CommandInput, Expected> {
  command: {
    contract: 'ocr2'
    id: 'inspect'
  }
  instructions: {
    contract: string
    function: string
  }[]
  makeInput: (flags: any, args: string[]) => Promise<InspectionInput<CommandInput, Expected>>
  makeOnchainData: (instructionsData: any[]) => Expected
  inspect: (expected: Expected, data: Expected) => boolean
}

export const instructionToInspectCommand = <CommandInput, Expected>(
  inspectInstruction: InspectInstruction<CommandInput, Expected>,
) => {
  const id = `${inspectInstruction.command.contract}:${inspectInstruction.command.id}`
  return class Command extends TerraCommand {
    static id = id
    command: AbstractCommand

    constructor(flags, args) {
      super(flags, args)
    }

    execute = async (): Promise<Result<TransactionResponse>> => {
      const input = await inspectInstruction.makeInput(this.flags, this.args)
      const commands = await Promise.all(
        inspectInstruction.instructions.map((instruction) =>
          makeAbstractCommand(
            `${instruction.contract}:${instruction.function}`,
            this.flags,
            this.args,
            input.commandInput,
          ),
        ),
      )

      const data = await Promise.all(
        commands.map(async (command) => {
          command.invokeMiddlewares(command, command.middlewares)
          const { data } = await command.execute()
          return data
        }),
      )

      const onchainData = inspectInstruction.makeOnchainData(data)
      const inspection = inspectInstruction.inspect(input.expected, onchainData)
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
