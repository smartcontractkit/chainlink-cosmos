import AbstractCommand, { makeAbstractCommand } from '.'
import { Result } from '@chainlink/gauntlet-core'
import { CosmosCommand, TransactionResponse, logger } from '@chainlink/gauntlet-cosmos'
import { prompt } from '@chainlink/gauntlet-core/dist/utils'
import { AccAddress } from '@chainlink/gauntlet-cosmos'
import { AccountData, OfflineSigner } from '@cosmjs/proto-signing'
import { SigningClient } from '@chainlink/gauntlet-cosmos/dist/commands/client'

export type ExecutionContext = {
  id: string
  contract: string
  wallet: OfflineSigner
  provider: SigningClient
  signer: AccountData
  flags: any
}

export type Input<UserInput, ContractInput> = {
  user: UserInput
  contract: ContractInput
}

export type BeforeExecute<UserInput, ContractInput> = (
  context: ExecutionContext,
  input: Input<UserInput, ContractInput>,
) => (signer: AccAddress) => Promise<void>

export type AfterExecute<UserInput, ContractInput> = (
  context: ExecutionContext,
  input: Input<UserInput, ContractInput>,
) => (response: Result<TransactionResponse>) => Promise<any>

export type ValidateFn<UserInput> = (input: UserInput, context: ExecutionContext) => Promise<boolean>
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
  validations?: {
    [id: string]: ValidateFn<UserInput>
  }
}

const defaultBeforeExecute = <UserInput, ContractInput>(
  context: ExecutionContext,
  input: Input<UserInput, ContractInput>,
) => async () => {
  logger.loading(`Executing ${context.id} from contract ${context.contract}`)
  logger.log('Input Params:', input.contract)
}

export const extendCommandInstruction = <UserInput, ContractInput>(
  instruction: AbstractInstruction<UserInput, ContractInput>,
  config: {
    suffixes: string[]
    validationsToSkip?: string[]
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
    validations: instruction.validations && filterValidations(instruction.validations, config.validationsToSkip),
  }
}

const filterValidations = <UserInput>(
  validations: { [id: string]: ValidateFn<UserInput> },
  toSkip?: string[],
): { [id: string]: ValidateFn<UserInput> } => {
  return Object.entries(validations).reduce((agg, [id, validate]) => {
    if (!toSkip?.includes(id)) return agg
    return { ...agg, ...{ [id]: validate } }
  }, {})
}

export const instructionToCommand = <UserInput, ContractInput>(
  instruction: AbstractInstruction<UserInput, ContractInput>,
) => {
  const id = `${instruction.instruction.contract}:${instruction.instruction.function}`
  const commandId = instruction.instruction.suffixes ? `${id}:${instruction.instruction.suffixes.join(':')}` : id
  const category = `${instruction.instruction.category}`
  const examples = instruction.examples || []

  return class Command extends CosmosCommand {
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

    runValidations = async (
      validations: { [id: string]: ValidateFn<UserInput> },
      executionContext: ExecutionContext,
      input: UserInput,
    ) => {
      logger.loading('Running command validations')
      const results = await Promise.all(
        Object.entries(validations).map(async ([id, validate]) => {
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
        if (!success) logger.error(msgFail)
        else logger.success(msgSuccess)
      })
      if (results.filter((r) => !r.success).length > 0) {
        throw new Error('Command validation failed')
      }
    }

    buildCommand = async (flags, args): Promise<CosmosCommand> => {
      const contract = args[0]

      const executionContext: ExecutionContext = {
        id,
        contract,
        provider: this.provider,
        wallet: this.wallet,
        signer: this.signer,
        flags,
      }

      const userInput = await instruction.makeInput(flags, args)

      // Validation
      if (instruction.validations) {
        const validationsToSkip = this.getValidationsToSkip(flags)
        const toValidate = filterValidations(instruction.validations, validationsToSkip)
        await this.runValidations(toValidate, executionContext, userInput)
      }

      // TODO: Some commands just provide a validateInput fn. Update those to give a set of validations
      if (!instruction.validateInput(userInput)) throw new Error(`Invalid input params: ${JSON.stringify(userInput)}`)

      const contractInput = await instruction.makeContractInput(userInput, executionContext)

      const input: Input<UserInput, ContractInput> = {
        user: userInput,
        contract: contractInput,
      }
      this.beforeExecute = instruction.beforeExecute
        ? instruction.beforeExecute(executionContext, input)
        : defaultBeforeExecute(executionContext, input)

      this.afterExecute = instruction.afterExecute
        ? instruction.afterExecute(executionContext, input)
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
      await this.command.simulateExecute_()
      await this.beforeExecute(this.signer.address)
      await prompt(`Continue?`)

      let response = await this.command.execute()
      const data = await this.afterExecute(response)
      return !!data ? { ...response, data: { ...data } } : response
    }
  }
}
