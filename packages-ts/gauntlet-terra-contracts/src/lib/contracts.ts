import { io, logger } from '@chainlink/gauntlet-core/dist/utils'
import { JSONSchemaType } from 'ajv'
import { existsSync, readFileSync } from 'fs'
import path from 'path'
import fetch from 'node-fetch'

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

export const loadContracts = (version): Contracts => {
  return Object.values(CONTRACT_LIST).reduce((agg, id) => {
    return {
      ...agg,
      ...{
        [id]: {
          id,
          abi: getContractABI(id),
          bytecode: getContractCode(id, version),
        },
      },
    }
  }, {} as Contracts)
}

export const getContractCode = async (contractId: CONTRACT_LIST, version): Promise<string> => {
  console.log(version)
  const response = await fetch(
    `https://github.com/smartcontractkit/chainlink-terra/releases/download/${version}/${contractId}.wasm`,
  )
  console.log(response)
  const body = await response.text()
  return body.toString(`base64`)
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
  return (contractId: CONTRACT_LIST, version): Contract => {
    // Preload contracts
    const contracts = loadContracts(version)
    return contracts[contractId]
  }
})()
