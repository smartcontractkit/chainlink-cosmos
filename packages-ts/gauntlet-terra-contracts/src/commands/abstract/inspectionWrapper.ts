import AbstractCommand, { makeAbstractCommand } from '.'
import { Result } from '@chainlink/gauntlet-core'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { CATEGORIES } from '../../lib/constants'
import { CONTRACT_LIST } from '../../lib/contracts'
import { APIParams } from '@terra-money/terra.js/dist/client/lcd/APIRequester'

export type Query = (contractAddress: string, query: any, params?: APIParams) => Promise<any>

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
    id: 'inspect'
  }
  instructions: {
    contract: string
    function: string
  }[]
  makeInput: (flags: any, args: string[]) => Promise<CommandInput>
  makeInspectionData: (query: Query) => (input: CommandInput) => Promise<ContractExpectedInfo>
  makeOnchainData: (
    query: Query,
  ) => (instructionsData: any[], input: CommandInput, contractAddress: string) => Promise<ContractExpectedInfo>
  inspect: (expected: ContractExpectedInfo, data: ContractExpectedInfo) => boolean
}

export const instructionToInspectCommand = <CommandInput, Expected>(
  inspectInstruction: InspectInstruction<CommandInput, Expected>,
) => {
  const id = `${inspectInstruction.command.contract}:${inspectInstruction.command.id}`
  return class Command extends TerraCommand {
    static id = id

    constructor(flags, args) {
      super(flags, args)
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

      const query: Query = this.provider.wasm.contractQuery.bind(this.provider.wasm)
      const onchainData = await inspectInstruction.makeOnchainData(query)(data, input, this.args[0])
      const inspectData = await inspectInstruction.makeInspectionData(query)(input)
      const inspection = inspectInstruction.inspect(inspectData, onchainData)
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
