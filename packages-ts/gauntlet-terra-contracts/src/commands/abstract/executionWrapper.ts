import { makeAbstractCommand } from '.'
import { Result } from '@chainlink/gauntlet-core'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { AccAddress } from '@terra-money/terra.js'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'

export type BeforeExecutionContext = {
  input: any
  contractInput: any
  commandId: string
  contract: string
}
export interface AbstractInstruction<Input, ContractInput> {
  examples?: string[]
  instruction: {
    category: string
    contract: string
    function: string
  }
  makeInput: (flags: any, args: string[]) => Promise<Input>
  validateInput: (input: Input) => boolean
  makeContractInput: (input: Input) => Promise<ContractInput>
  beforeExecute?: (context: BeforeExecutionContext) => Promise<void>
  afterExecute?: (response: Result<TransactionResponse>) => any
}

export const defaultAfterExecute = async (response: Result<TransactionResponse>): Promise<void> => {
  logger.success(`Execution finished at transaction: ${response.responses[0].tx.hash}`)
}

export const defaultBeforeExecute = async (context: BeforeExecutionContext) => {
  logger.loading(`Executing ${context.commandId} from contract ${context.contract}`)
  logger.log('Input Params:', context.contractInput)
  await prompt(`Continue?`)
}

export const instructionToCommand = <Input, ContractInput>(instruction: AbstractInstruction<Input, ContractInput>) => {
  const id = `${instruction.instruction.contract}:${instruction.instruction.function}`
  const category = `${instruction.instruction.category}`
  const examples = instruction.examples || []
  return class Command extends TerraCommand {
    static id = id
    static category = category
    static examples = examples

    input: Input
    contractInput: ContractInput

    constructor(flags, args) {
      super(flags, args)
    }

    beforeExecute = instruction.beforeExecute || defaultBeforeExecute
    afterExecute = instruction.afterExecute || defaultAfterExecute

    buildCommand = async (): Promise<TerraCommand> => {
      this.input = await instruction.makeInput(this.flags, this.args)
      if (!instruction.validateInput(this.input)) {
        throw new Error(`Invalid input params:  ${JSON.stringify(this.input)}`)
      }
      this.contractInput = await instruction.makeContractInput(this.input)
      const abstractCommand = await makeAbstractCommand(id, this.flags, this.args, this.contractInput)
      await abstractCommand.invokeMiddlewares(abstractCommand, abstractCommand.middlewares)
      return abstractCommand
    }

    makeRawTransaction = async (signer: AccAddress) => {
      const command = await this.buildCommand()
      return command.makeRawTransaction(signer)
    }

    execute = async (): Promise<Result<TransactionResponse>> => {
      const command = await this.buildCommand()
      await this.beforeExecute({
        input: this.input,
        contractInput: this.contractInput,
        commandId: id,
        contract: this.args[0],
      })
      let response = await command.execute()
      const data = this.afterExecute(response)
      if (data) {
        response = { ...response, data: { ...data } }
      }
      return response
    }
  }
}
