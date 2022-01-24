import { io, logger } from '@chainlink/gauntlet-core/dist/utils'
import { JSONSchemaType } from 'ajv'
import { existsSync, readFileSync } from 'fs'
import path from 'path'

export enum CONTRACT_LIST {
  FLAGS = 'flags',
  DEVIATION_FLAGGING_VALIDATOR = 'deviation_flagging_validator',
  OCR_2 = 'ocr2',
  ACCESS_CONTROLLER = 'access_controller',
}

export enum TERRA_OPERATIONS {
  DEPLOY = 'instantiate',
  EXECUTE = 'execute',
  QUERY = 'query',
}

export type TerraABI = {
  [TERRA_OPERATIONS.DEPLOY]: JSONSchemaType<any>
  [TERRA_OPERATIONS.EXECUTE]: JSONSchemaType<any>
  [TERRA_OPERATIONS.QUERY]: JSONSchemaType<any>
}

export type Contract = {
  id: CONTRACT_LIST
  abi: TerraABI
  bytecode: string
}

export type Contracts = Record<CONTRACT_LIST, Contract>

export const loadContracts = (): Contracts => {
  return Object.values(CONTRACT_LIST).reduce((agg, id) => {
    return {
      ...agg,
      ...{
        [id]: {
          id,
          abi: getContractABI(id),
          bytecode: getContractCode(id),
        },
      },
    }
  }, {} as Contracts)
}

// TODO: Pull it from a Github versioned release artifact
export const getContractCode = (contractId: CONTRACT_LIST): string => {
  // Possible paths depending on how/where gauntlet is being executed
  const possibleContractPaths = [
    path.join(__dirname, '../..', './artifacts/bin'),
    path.join(process.cwd(), './artifacts/bin'),
    path.join(process.cwd(), './packages-ts/gauntlet-terra-contracts/artifacts/bin'),
  ]
  const codes = possibleContractPaths
    .filter((contractPath) => existsSync(`${contractPath}/${contractId}.wasm`))
    .map((contractPath) => {
      const wasm = readFileSync(`${contractPath}/${contractId}.wasm`)
      return wasm.toString('base64')
    })
  return codes[0]
}

const contractDirName = {
  [CONTRACT_LIST.FLAGS]: 'flags',
  [CONTRACT_LIST.DEVIATION_FLAGGING_VALIDATOR]: 'deviation-flagging-validator',
  [CONTRACT_LIST.OCR_2]: 'ocr2',
  [CONTRACT_LIST.ACCESS_CONTROLLER]: 'access-controller',
}

export const getContractABI = (contractId: CONTRACT_LIST): TerraABI => {
  // Possible paths depending on how/where gauntlet is being executed
  const possibleContractPaths = [
    path.join(__dirname, '../../../..', './contracts'),
    path.join(process.cwd(), './contracts'),
    path.join(process.cwd(), '../..', './contracts'),
  ]

  const toDirName = (contractId: CONTRACT_LIST) => contractDirName[contractId]

  const abi = possibleContractPaths
    .filter((path) => existsSync(`${path}/${toDirName(contractId)}/schema`))
    .map((contractPath) => {
      const toPath = (type) => path.join(contractPath, `./${toDirName(contractId)}/schema/${type}`)
      return {
        execute: io.readJSON(toPath('execute_msg')),
        query: io.readJSON(toPath('query_msg')),
        instantiate: io.readJSON(toPath('instantiate_msg')),
      }
    })
  if (abi.length === 0) {
    logger.error(`ABI not found for contract ${contractId}`)
  }

  return abi[0]
}

export const getContract = (() => {
  // Preload contracts
  const contracts = loadContracts()
  return (contractId: CONTRACT_LIST): Contract => contracts[contractId]
})()
