import AbstractCommand, { makeAbstractCommand } from '.'
import { Result } from '@chainlink/gauntlet-core'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'

export interface AbstractInstruction<Input, ContractInput> {
  instruction: {
    contract: string
    function: string
  }
  makeInput: (flags: any, args: string[]) => Promise<Input>
  validateInput: (input: Input) => boolean
  makeContractInput: (input: Input) => Promise<ContractInput>
}

export const instructionToCommand = (instruction: AbstractInstruction<any, any>) => {
  const id = `${instruction.instruction.contract}:${instruction.instruction.function}`
  return class Command extends TerraCommand {
    static id = id
    command: AbstractCommand

    constructor(flags, args) {
      super(flags, args)
    }

    execute = async (): Promise<Result<TransactionResponse>> => {
      const commandInput = await instruction.makeInput(this.flags, this.args)
      instruction.validateInput(commandInput)
      const input = await instruction.makeContractInput(commandInput)
      const abstractCommand = await makeAbstractCommand(id, this.flags, this.args, input)
      abstractCommand.invokeMiddlewares(abstractCommand, abstractCommand.middlewares)
      return abstractCommand.execute()
    }
  }
}

export interface InspectInstruction<Input, OnchainData> {
  instruction: {
    contract: string
    function: string
  }
  inspect: (input: Input, data: OnchainData) => boolean
  makeInput: (flags: any) => Promise<Input>
}

export const instructionToInspectCommand = <Input, OnchainData>(
  inspectInstruction: InspectInstruction<Input, OnchainData>,
) => {
  const id = `${inspectInstruction.instruction.contract}:inspect`
  return class Command extends TerraCommand {
    static id = id
    command: AbstractCommand

    constructor(flags, args) {
      super(flags, args)
    }

    execute = async (): Promise<Result<TransactionResponse>> => {
      const abstractCommand = await makeAbstractCommand(id, this.flags, this.args)
      abstractCommand.invokeMiddlewares(abstractCommand, abstractCommand.middlewares)

      const generatedData = await inspectInstruction.makeInput(this.flags)
      const { data } = await abstractCommand.execute()
      const inspection = inspectInstruction.inspect(generatedData, data)
      return {
        data: inspection,
        responses: [
          {
            contract: this.args[0],
          },
        ],
      } as Result<TransactionResponse>
    }
  }
}
