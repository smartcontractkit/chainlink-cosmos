import AbstractCommand, { makeAbstractCommand } from '.'
import { Result } from '@chainlink/gauntlet-core'
import { TerraCommand, TransactionResponse, logger } from '@chainlink/gauntlet-terra'
import { prompt } from '@chainlink/gauntlet-core/dist/utils'
import { AccAddress, LCDClient, Wallet } from '@terra-money/terra.js'

export type ExecutionContext = {
  id: string
  contract: string
  wallet: Wallet
  provider: LCDClient
  flags: any
}

export type InputContext<Input, ContractInput> = {
  input: Input
  contractInput: ContractInput
}

export type BeforeExecute<Input, ContractInput> = (
  context: ExecutionContext,
  inputContext: InputContext<Input, ContractInput>,
) => (signer: AccAddress) => Promise<void>

export type AfterExecute<Input, ContractInput> = (
  context: ExecutionContext,
  inputContext: InputContext<Input, ContractInput>,
) => (response: Result<TransactionResponse>) => Promise<any>

export interface Validation<Input> {
  id: string
  msgSuccess: string
  msgFail: string
  validate: (context: ExecutionContext) => (input: Input) => Promise<boolean>
}

export interface AbstractInstruction<Input, ContractInput> {
  examples?: string[]
  instruction: {
    category: string
    contract: string
    function: string
    subInstruction?: string
  }
  makeInput: (flags: any, args: string[]) => Promise<Input>
  validateInput: (input: Input) => boolean
  makeContractInput: (input: Input, context: ExecutionContext) => Promise<ContractInput>
  beforeExecute?: BeforeExecute<Input, ContractInput>
  afterExecute?: AfterExecute<Input, ContractInput>
  validations?: Validation<Input>[]
}

const defaultBeforeExecute = <Input, ContractInput>(
  context: ExecutionContext,
  inputContext: InputContext<Input, ContractInput>,
) => async () => {
  logger.loading(`Executing ${context.id} from contract ${context.contract}`)
  logger.log('Input Params:', inputContext.contractInput)
  await prompt(`Continue?`)
}

export const instructionToCommand = <Input, ContractInput>(instruction: AbstractInstruction<Input, ContractInput>) => {
  const id = `${instruction.instruction.contract}:${instruction.instruction.function}`
  const commandId = instruction.instruction.subInstruction ? `${id}:${instruction.instruction.subInstruction}` : id
  const category = `${instruction.instruction.category}`
  const examples = instruction.examples || []

  return class Command extends TerraCommand {
    static id = commandId
    static category = category
    static examples = examples

    command: AbstractCommand

    constructor(flags, args) {
      super(flags, args)
    }

    getValidationsToSkip = (flags: any): string[] => {
      return Object.keys(flags)
        .filter((key) => key.startsWith('skip-'))
        .map((value) => value.replace('skip-', ''))
    }

    validateContractAddress = (address: string) => {
      if (!AccAddress.validate(address)) throw new Error(`Invalid contract address ${address}`)
    }

    runValidations = async (validations: Validation<Input>[], executionContext: ExecutionContext, input: Input) => {
      logger.loading('Running command validations')
      const results = await Promise.all(
        validations.map(async ({ validate, msgFail, msgSuccess }) => {
          try {
            return { success: await validate(executionContext)(input), msgFail, msgSuccess }
          } catch (e) {
            return { success: false, msgFail: e.message || msgFail, msgSuccess }
          }
        }),
      )
      results.forEach(({ success, msgFail, msgSuccess }) => {
        if (!success) logger.error(`Validation Failed: ${msgFail}`)
        else logger.success(`Validation Succeeded: ${msgSuccess}`)
      })
      if (results.filter((r) => !r.success).length > 0) {
        throw new Error('Command validation failed')
      }
    }

    buildCommand = async (flags, args): Promise<TerraCommand> => {
      const contract = args[0]

      const executionContext: ExecutionContext = {
        id,
        contract,
        provider: this.provider,
        wallet: this.wallet,
        flags,
      }

      const input = await instruction.makeInput(flags, args)

      // Validation
      if (instruction.validations) {
        const validationsToSkip = this.getValidationsToSkip(flags)
        const validations = instruction.validations.filter(({ id }) => !validationsToSkip.includes(id))
        await this.runValidations(validations, executionContext, input)
      }
      if (!instruction.validateInput(input)) throw new Error(`Invalid input params: ${JSON.stringify(input)}`)

      const contractInput = await instruction.makeContractInput(input, executionContext)

      const inputContext: InputContext<Input, ContractInput> = {
        input,
        contractInput,
      }
      this.beforeExecute = instruction.beforeExecute
        ? instruction.beforeExecute(executionContext, inputContext)
        : defaultBeforeExecute(executionContext, inputContext)

      this.afterExecute = instruction.afterExecute
        ? instruction.afterExecute(executionContext, inputContext)
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

export interface TailInstruction<Input, InnerCommandInput> {
  command: typeof TerraCommand
  ui: {
    suffixes: string[]
    examples: string[]
  }
  makeInput: (flags, args) => Input
  makeInnerCommandInput: (input: Input) => InnerCommandInput
  skipValidations: string[]
}

export const instructionToTailCommand = <Input, AcceptProposalInput>(
  instruction: TailInstruction<Input, AcceptProposalInput>,
) => {
  const id = `${instruction.command.id}:${instruction.ui.suffixes.join(':')}`
  return class SuperCommand extends TerraCommand {
    static id = id
    static category = instruction.command.category

    innerCommand: TerraCommand

    constructor(flags, args) {
      super(flags, args)
    }

    makeRawTransaction = (signer) => {
      return this.innerCommand.makeRawTransaction(signer)
    }

    buildCommand = async (flags, args) => {
      const input = instruction.makeInput(flags, args)
      const innerInput = { input: instruction.makeInnerCommandInput(input) }
      const skipFlags = instruction.skipValidations.reduce(
        (agg, validation) => ({ ...agg, [`skip-${validation}`]: true }),
        {},
      )
      const innerFlags = { ...flags, ...skipFlags, ...innerInput }

      this.innerCommand = new (instruction.command as any)(innerFlags, args) as TerraCommand
      this.innerCommand.invokeMiddlewares(this.innerCommand, this.innerCommand.middlewares)

      return this
    }

    execute = async () => {
      await this.buildCommand(this.flags, this.args)
      return this.innerCommand.execute()
    }
  }
}
