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
