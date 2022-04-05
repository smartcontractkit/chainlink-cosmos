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

export type Input<UserInput, ContractInput> = {
  input: UserInput
  contractInput: ContractInput
}

export type BeforeExecute<UserInput, ContractInput> = (
  context: ExecutionContext,
  inputContext: Input<UserInput, ContractInput>,
) => (signer: AccAddress) => Promise<void>

export type AfterExecute<UserInput, ContractInput> = (
  context: ExecutionContext,
  inputContext: Input<UserInput, ContractInput>,
) => (response: Result<TransactionResponse>) => Promise<any>

export type Validate<UserInput> = (input: UserInput, context: ExecutionContext) => Promise<boolean>
export interface Validation<UserInput> {
  id: number
  validate: Validate<UserInput>
}

export const makeValidations = (...validates: Validate<any>[]): Validation<any>[] =>
  validates.map((validate, idx) => ({ id: idx, validate }))

export interface AbstractInstruction<UserInput, ContractInput> {
  examples?: string[]
  instruction: {
    category: string
    contract: string
    function: string
    suffixes?: string[]
  }
  makeInput: (flags: any, args: string[]) => Promise<UserInput>
  validateInput: (input: UserInput) => boolean
  makeContractInput: (input: UserInput, context: ExecutionContext) => Promise<ContractInput>
  beforeExecute?: BeforeExecute<UserInput, ContractInput>
  afterExecute?: AfterExecute<UserInput, ContractInput>
  validations?: Validation<UserInput>[]
}

const defaultBeforeExecute = <UserInput, ContractInput>(
  context: ExecutionContext,
  inputContext: Input<UserInput, ContractInput>,
) => async () => {
  logger.loading(`Executing ${context.id} from contract ${context.contract}`)
  logger.log('Input Params:', inputContext.contractInput)
  await prompt(`Continue?`)
}

export const extendCommandInstruction = <UserInput, ContractInput>(
  instruction: AbstractInstruction<UserInput, ContractInput>,
  config: {
    suffixes: string[]
    validationsToSkip?: number[]
    makeInput: (flags: any, args: string[]) => Promise<UserInput>
    examples: string[]
  },
): AbstractInstruction<UserInput, ContractInput> => {
  return {
    ...instruction,
    examples: config.examples || instruction.examples,
    instruction: {
      ...instruction.instruction,
      suffixes: config.suffixes,
    },
    makeInput: config.makeInput,
    validations:
      instruction.validations && instruction.validations.filter(({ id }) => !config.validationsToSkip?.includes(id)),
  }
}

export const instructionToCommand = <UserInput, ContractInput>(
  instruction: AbstractInstruction<UserInput, ContractInput>,
) => {
  const id = `${instruction.instruction.contract}:${instruction.instruction.function}`
  const commandId = instruction.instruction.suffixes ? `${id}:${instruction.instruction.suffixes.join(':')}` : id
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

    getValidationsToSkip = (flags: any): number[] => {
      return Object.keys(flags)
        .filter((key) => key.startsWith('skip-'))
        .map((value) => Number(value.replace('skip-', '')))
    }

    validateContractAddress = (address: string) => {
      if (!AccAddress.validate(address)) throw new Error(`Invalid contract address ${address}`)
    }

    runValidations = async (
      validations: Validation<UserInput>[],
      executionContext: ExecutionContext,
      input: UserInput,
    ) => {
      logger.loading('Running command validations')
      const results = await Promise.all(
        validations.map(async ({ validate, id }) => {
          try {
            return {
              success: await validate(input, executionContext),
              msgFail: `Validation ${id} Failed`,
              msgSuccess: `Validation ${id} Succeeded`,
            }
          } catch (e) {
            return { success: false, msgFail: e.message, msgSuccess: '' }
          }
        }),
      )
      results.forEach(({ success, msgFail, msgSuccess }) => {
        if (!success) logger.error(`Validation Failed: ${msgFail}`)
        else logger.success(msgSuccess)
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
        const toValidate = instruction.validations.filter(({ id }) => !validationsToSkip.includes(id))
        await this.runValidations(toValidate, executionContext, input)
      }
      if (!instruction.validateInput(input)) throw new Error(`Invalid input params: ${JSON.stringify(input)}`)

      const contractInput = await instruction.makeContractInput(input, executionContext)

      const inputContext: Input<UserInput, ContractInput> = {
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
