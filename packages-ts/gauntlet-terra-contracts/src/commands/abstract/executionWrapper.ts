import AbstractCommand, { makeAbstractCommand } from '.'
import { Result } from '@chainlink/gauntlet-core'
import { TerraCommand, TransactionResponse, logger } from '@chainlink/gauntlet-terra'
import { AccAddress, LCDClient } from '@terra-money/terra.js'
import { prompt } from '@chainlink/gauntlet-core/dist/utils'

export type ExecutionContext<Input, ContractInput> = {
  input: Input
  contractInput: ContractInput
  id: string
  contract: string
  provider: LCDClient
  flags: any
}

export type BeforeExecute<Input, ContractInput> = (
  context: ExecutionContext<Input, ContractInput>,
) => (signer: AccAddress) => Promise<void>

export type AfterExecute<Input, ContractInput> = (
  context: ExecutionContext<Input, ContractInput>,
) => (response: Result<TransactionResponse>) => Promise<any>

export type BatchExecutionContext<Input, ContractInput> = {
  inputs: Input[]
  contractInputs: ContractInput[]
  id: string
  contract: string
  provider: LCDClient
  flags: any
}

export type BatchBeforeExecute<Input, ContractInput> = (
  context: BatchExecutionContext<Input, ContractInput>,
) => (signer: AccAddress) => Promise<void>

export type BatchAfterExecute<Input, ContractInput> = (
  context: BatchExecutionContext<Input, ContractInput>,
) => (response: Result<TransactionResponse>) => Promise<any>
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
  beforeExecute?: BeforeExecute<Input, ContractInput> | BatchBeforeExecute<Input, ContractInput>
  afterExecute?: AfterExecute<Input, ContractInput> | BatchAfterExecute<Input, ContractInput>
}

const defaultBeforeExecute = <Input, ContractInput>(context: ExecutionContext<Input, ContractInput>) => async () => {
  logger.loading(`Executing ${context.id} from contract ${context.contract}`)
  logger.log('Input Params:', context.contractInput)
  await prompt(`Continue?`)
}

const defaultBeforeBatchExecute = <Input, ContractInput>(
  context: BatchExecutionContext<Input, ContractInput>,
) => async () => {
  logger.loading(`Executing ${context.id} from contract ${context.contract} for the following sets of inputs`)
  context.contractInputs.forEach((element) => logger.log('Input Params:', element))
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

    command: AbstractCommand

    constructor(flags, args) {
      super(flags, args)
    }

    buildCommand = async (flags, args): Promise<TerraCommand> => {
      const input = await instruction.makeInput(flags, args)
      if (!instruction.validateInput(input)) {
        throw new Error(`Invalid input params:  ${JSON.stringify(input)}`)
      }
      const contractInput = await instruction.makeContractInput(input)
      const executionContext: ExecutionContext<Input, ContractInput> = {
        input,
        contractInput,
        id,
        contract: this.args[0],
        provider: this.provider,
        flags,
      }
      this.beforeExecute = instruction.beforeExecute
        ? (instruction.beforeExecute as BeforeExecute<Input, ContractInput>)(executionContext)
        : defaultBeforeExecute(executionContext)

      this.afterExecute = instruction.afterExecute
        ? (instruction.afterExecute as AfterExecute<Input, ContractInput>)(executionContext)
        : this.afterExecute

      const abstractCommand = await makeAbstractCommand(id, this.flags, this.args, contractInput)
      await abstractCommand.invokeMiddlewares(abstractCommand, abstractCommand.middlewares)
      this.command = abstractCommand

      return this
    }

    makeRawTransaction = async (signer: AccAddress) => {
      return this.command.makeRawTransaction(signer)
    }

    execute = async (): Promise<Result<TransactionResponse>> => {
      // TODO: Command should be built from gauntet-core
      await this.buildCommand(this.flags, this.args)
      await this.command.simulateExecute()
      await this.beforeExecute(this.wallet.key.accAddress)

      let response = await this.command.execute()
      const data = await this.afterExecute(response)
      return !!data ? { ...response, data: { ...data } } : response
    }
  }
}

export const instructionToBatchCommand = <Input, ContractInput>(
  instruction: AbstractInstruction<Input, ContractInput>,
) => {
  const id = `${instruction.instruction.contract}:${instruction.instruction.function}:batch`
  const category = `${instruction.instruction.category}`
  const examples = [] // TODO: Pass in accurate batch examples

  return class Command extends TerraCommand {
    static id = id
    static category = category
    static examples = examples

    command: AbstractCommand

    constructor(flags, args) {
      super(flags, args)
    }

    buildCommand = async (flags, args): Promise<TerraCommand> => {
      var inputs: Input[] = []
      var contractInputs: ContractInput[] = []

      flags.input.forEach(async (element) => {
        const input = await instruction.makeInput(element, args)
        if (!instruction.validateInput(input)) {
          throw new Error(`Invalid input params:  ${JSON.stringify(input)}`)
        }
        const contractInput = await instruction.makeContractInput(input)

        inputs.push(input)
        contractInputs.push(contractInput)
      })

      const executionContext: BatchExecutionContext<Input, ContractInput> = {
        inputs,
        contractInputs,
        id,
        contract: this.args[0],
        provider: this.provider,
        flags,
      }
      this.beforeExecute = instruction.beforeExecute
        ? (instruction.beforeExecute as BatchBeforeExecute<Input, ContractInput>)(executionContext)
        : defaultBeforeBatchExecute(executionContext)

      this.afterExecute = instruction.afterExecute
        ? (instruction.afterExecute as BatchAfterExecute<Input, ContractInput>)(executionContext)
        : this.afterExecute

      const abstractCommand = await makeAbstractCommand(id, this.flags, this.args, contractInputs)
      await abstractCommand.invokeMiddlewares(abstractCommand, abstractCommand.middlewares)
      this.command = abstractCommand

      return this
    }

    makeRawTransaction = async (signer: AccAddress) => {
      return this.command.makeRawTransaction(signer)
    }

    execute = async (): Promise<Result<TransactionResponse>> => {
      // TODO: Command should be built from gauntet-core
      await this.buildCommand(this.flags, this.args)
      await this.command.simulateExecute()
      await this.beforeExecute(this.wallet.key.accAddress)

      let response = await this.command.execute()
      const data = await this.afterExecute(response)
      return !!data ? { ...response, data: { ...data } } : response
    }
  }
}
