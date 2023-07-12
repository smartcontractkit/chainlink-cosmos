import { Result } from '@chainlink/gauntlet-core'
import { AccAddress } from '@chainlink/gauntlet-cosmos'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { TransactionResponse, CosmosCommand } from '@chainlink/gauntlet-cosmos'
import { Contract, CONTRACT_LIST, contracts, CosmosABI, COSMOS_OPERATIONS } from '../../lib/contracts'
// import { DEFAULT_RELEASE_VERSION } from '../../lib/constants'
import schema from '../../lib/schema'
import { withAddressBook } from '../../lib/middlewares'
import { MsgExecuteContract } from 'cosmjs-types/cosmwasm/wasm/v1/tx'
import { MsgExecuteContractEncodeObject } from '@cosmjs/cosmwasm-stargate'
import { EncodeObject } from '@cosmjs/proto-signing'
import { toUtf8 } from '@cosmjs/encoding'

export interface AbstractOpts {
  contract: Contract
  function: string
  action: COSMOS_OPERATIONS.DEPLOY | COSMOS_OPERATIONS.EXECUTE | COSMOS_OPERATIONS.QUERY | 'help'
}

export interface AbstractParams {
  [param: string]: any
}

export const makeAbstractCommand = async (
  instruction: string,
  flags: any,
  args: string[],
  input?: any,
): Promise<AbstractCommand> => {
  const commandOpts = await parseInstruction(instruction, flags.version)
  const params = parseParams(commandOpts, input || flags)
  return new AbstractCommand(flags, args, commandOpts, params)
}

export const parseInstruction = async (instruction: string, inputVersion: string): Promise<AbstractOpts> => {
  const isValidFunction = (abi: CosmosABI, functionName: string): boolean => {
    // Check against ABI if method exists
    const availableFunctions = [
      ...(abi.query.oneOf || abi.query.anyOf || []),
      ...(abi.execute.oneOf || abi.execute.anyOf || []),
    ].reduce((agg, prop) => {
      if (prop?.required && prop.required.length > 0) return [...agg, ...prop.required]
      if (prop?.enum && prop.enum.length > 0) return [...agg, ...prop.enum]
      return [...agg]
    }, [])
    logger.debug(`Available functions on this contract: ${availableFunctions}`)
    return availableFunctions.includes(functionName)
  }

  const isQueryFunction = (abi: CosmosABI, functionName: string) => {
    const functionList = abi.query.oneOf || abi.query.anyOf
    return functionList.find((queryAbi: any) => {
      if (queryAbi.enum) return queryAbi.enum.includes(functionName)
      if (queryAbi.required) return queryAbi.required.includes(functionName)
      return false
    })
  }

  const command = instruction.split(':')
  if (!command.length || command.length > 2) throw new Error(`Abstract: Instruction ${command.join(':')} not found`)

  const id = command[0] as CONTRACT_LIST
  const contract = await contracts.getContractWithSchemaAndCode(id, inputVersion)
  if (!contract) throw new Error(`Abstract: Contract ${command[0]} not found`)

  if (command[1] === 'help') {
    return {
      contract,
      function: 'help',
      action: 'help',
    }
  }

  if (command[1] === 'deploy') {
    return {
      contract,
      function: COSMOS_OPERATIONS.DEPLOY,
      action: COSMOS_OPERATIONS.DEPLOY,
    }
  }

  const functionName = isValidFunction(contract.abi, command[1]) && command[1]
  if (!functionName) throw new Error(`Abstract: Function ${command[1]} for contract ${contract.id} not found`)

  return {
    contract,
    function: functionName,
    action: isQueryFunction(contract.abi, functionName) ? COSMOS_OPERATIONS.QUERY : COSMOS_OPERATIONS.EXECUTE,
  }
}

export const parseParams = (commandOpts: AbstractOpts, params: any): AbstractParams => {
  if (commandOpts.action === 'help') return params
  const abiSchema = commandOpts.contract.abi[commandOpts.action]
  const validate = schema.compile(abiSchema)
  const schemas = [...(abiSchema.oneOf || []), ...(abiSchema.anyOf || [])]
  // isEnum means the function doesn't receive any parameter
  const isEnumData = schemas.some((subSchema) => !!subSchema.enum && subSchema.enum.includes(commandOpts.function))
  const data = isEnumData
    ? commandOpts.function
    : commandOpts.action === COSMOS_OPERATIONS.DEPLOY
    ? params
    : {
        [commandOpts.function]: params,
      }

  if (!validate(data)) {
    logger.log(validate.errors)
    throw new Error(`Error validating parameters for function ${commandOpts.function}`)
  }

  return data
}

type AbstractExecute = (params: any, address?: string) => Promise<Result<TransactionResponse>>
export default class AbstractCommand extends CosmosCommand {
  opts: AbstractOpts
  params: AbstractParams

  constructor(flags, args, opts, params) {
    super(flags, args)
    this.use(withAddressBook)
    this.opts = opts
    this.params = params

    if ([...COSMOS_OPERATIONS.EXECUTE, ...COSMOS_OPERATIONS.QUERY].includes(this.opts.action)) {
      this.require(args[0], 'Provide a valid contract address')
    }

    this.contracts = [this.opts.contract.id]
  }

  makeRawTransaction = async (signer: AccAddress): Promise<EncodeObject[]> => {
    const contractAddress = this.args[0]
    const msg: MsgExecuteContractEncodeObject = {
      typeUrl: '/cosmwasm.wasm.v1.MsgExecuteContract',
      value: MsgExecuteContract.fromPartial({
        sender: signer,
        contract: contractAddress,
        msg: toUtf8(JSON.stringify(this.params)),
        funds: [],
      }),
    }
    return [msg]
  }

  abstractDeploy: AbstractExecute = async (params: any) => {
    logger.loading(`Deploying contract ${this.opts.contract.id}`)
    const codeId = this.codeIds[this.opts.contract.id]
    this.require(!!codeId, `Code Id for contract ${this.opts.contract.id} not found`)
    const deploy = await this.deploy(codeId, params)
    logger.success(`Deployed ${this.opts.contract.id} to ${deploy.contractAddress}`)
    return {
      responses: [
        {
          tx: deploy,
          contract: deploy.contractAddress,
        },
      ],
    } as Result<any>
  }

  abstractExecute: AbstractExecute = async (params: any, address: string) => {
    logger.debug(`Executing ${this.opts.function} from contract ${this.opts.contract.id} at ${address}`)
    const tx = await this.call(address, params)
    logger.debug(`Execution finished at tx ${tx.transactionHash}`)
    return {
      responses: [
        {
          tx,
          contract: address,
        },
      ],
    } as Result<any>
  }

  abstractQuery: AbstractExecute = async (params: any, address: string) => {
    logger.debug(`Calling ${this.opts.function} from contract ${this.opts.contract.id} at ${address}`)
    const result = await this.provider.queryContractSmart(address, params)
    logger.debug(`Query finished with result: ${JSON.stringify(result)}`)
    return {
      data: result,
      responses: [
        {
          contract: address,
        },
      ],
    } as Result<any>
  }

  abstractHelp: AbstractExecute = async () => {
    // TODO: Get functions per operation with their own needed parameters. AJV doesn't offer a way to do this
    // const queryFunctions = getSchemaFunctions(this.opts.contract.abi.query)
    // const deployFunction = getSchemaFunctions(this.opts.contract.abi.instantiate)
    // const executeFunctions = getSchemaFunctions(this.opts.contract.abi.execute)
    return {
      responses: [],
    }
  }

  simulateExecute_ = async () => {
    if (this.opts.action !== COSMOS_OPERATIONS.EXECUTE) {
      logger.info('Skipping tx simulation for non-execute operation')
      return
    }

    const contractAddress = this.args[0]
    logger.loading(`Executing tx simulation for ${this.opts.contract.id}:${this.opts.function} at ${contractAddress}`)
    const estimatedGas = await this.simulateExecute(contractAddress, [this.params])
    logger.info(`Tx simulation successful: estimated gas usage is ${estimatedGas}`)
    return estimatedGas
  }

  execute = async () => {
    const operations = {
      [COSMOS_OPERATIONS.DEPLOY]: this.abstractDeploy,
      [COSMOS_OPERATIONS.QUERY]: this.abstractQuery,
      [COSMOS_OPERATIONS.EXECUTE]: this.abstractExecute,
      help: this.abstractHelp,
    }

    const address = this.args[0]
    return operations[this.opts.action](this.params, address)
  }
}
