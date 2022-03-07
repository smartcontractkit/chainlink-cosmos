import AbstractCommand, { makeAbstractCommand } from '.'
import { Result } from '@chainlink/gauntlet-core'
import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { AccAddress, LCDClient } from '@terra-money/terra.js'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { parse as parseCSV } from 'csv-parse/sync'
import { readFileSync } from 'fs'

export type ExecutionContext<Input, ContractInput> = {
  input: Input
  contractInput: ContractInput
  id: string
  contract: string
  provider: LCDClient
  flags: any
}

export type BatchExecutionContext<Input, ContractInput> = {
  inputs: Input[]
  contractInputs: ContractInput[]
  id: string
  contract: string
  provider: LCDClient
  flags: any
}

export type BeforeExecute<Input, ContractInput> = (
  context: ExecutionContext<Input, ContractInput>,
) => (signer: AccAddress) => Promise<void>

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
  beforeExecute?: BeforeExecute<Input, ContractInput>
  afterExecute?: (response: Result<TransactionResponse>) => any
}

const defaultBeforeExecute = <Input, ContractInput>(context: ExecutionContext<Input, ContractInput>) => async () => {
  logger.loading(`Executing ${context.id} from contract ${context.contract}`)
  logger.log('Input Params:', context.contractInput)
  await prompt(`Continue?`)
}

const defaultBeforeBatchExecute = <Input, ContractInput>(context: BatchExecutionContext<Input, ContractInput>) => async () => {
  logger.loading(`Executing ${context.id} from contract ${context.contract}`)
  logger.log('Input Params:')
  for (int i; i < context.contractInputs; i++) {
    logger.log(`${context.contractInputs}`)
  }
  await prompt(`Continue?`)
}

const readBatchFile = async (batchFile: string): Promise<any[]> => {
    const content = readFileSync(batchFile, 'utf8')
    const parseContent = {
      'json': JSON.parse,
      'csv': (content) => {
        parseCSV(content, {
          columns: true, skip_empty_lines: true
        })
      }
    }
    const inputs = parseContent[batchFile.split('.').pop() as string](content)

    if (!Array.isArray(inputs)) {
      throw Error(`Invalid contents of ${batchFile}: ${inputs}`)
    } else if (inputs.length == 0) {
      throw Error(`Read empty transfer list from ${batchFile}, aborting.`)
    }
    return inputs
}

//const makeContractInputs = <ContractInput, Input>async(inputs: Input[], makeContractInput: async (Input) => Promise<ContractInput>) : Promise<ContractInput[]> => {
//  return await inputs.map(async (inp) => await makeContractInput(inp))
//}

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

    afterExecute = instruction.afterExecute || this.afterExecute

    buildBatchCommand = async (flags, args): Promise<TerraCommand> => {
      const inputs = await readBatchFile(flags.batch)

      if (!inputs.map(instruction.validateInput)) {
        throw new Error(`Invalid batch inputs:  ${JSON.stringify(inputs)}`)
      }

      const contractInputs = await makeContractInputs<ContractInput>(inputs, instruction.makeContractInput)
      const executionContext: BatchExecutionContext<Input, ContractInput> = {
        inputs,
        contractInputs,
        id,
        contract: this.args[0],
        provider: this.provider,
        flags,
      }
      this.beforeBatchExecute = instruction.beforeExecute
        ? instruction.beforeExecute(executionContext)
        : defaultBeforeBatchExecute(executionContext)

      const abstractCommand = await makeAbstractCommand(id, this.flags, this.args, contractInputs)
      await abstractCommand.invokeMiddlewares(abstractCommand, abstractCommand.middlewares)
      this.command = abstractCommand

      return this
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
        ? instruction.beforeExecute(executionContext)
        : defaultBeforeExecute(executionContext)

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
      if (this.flags.batch) {
        await this.buildCommand(this.flags, this.args)
      } else {
        await this.buildBatchCommand(this.flags, this.args)
      }
      await this.command.simulateExecute()
      await this.beforeExecute(this.wallet.key.accAddress)

      let response = await this.command.execute()
      const data = this.afterExecute(response)
      return !!data ? { ...response, data: { ...data } } : response
    }
  }
}
