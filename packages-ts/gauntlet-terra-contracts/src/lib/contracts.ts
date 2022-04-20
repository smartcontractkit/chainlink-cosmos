import { io, logger } from '@chainlink/gauntlet-core/dist/utils'
import { JSONSchemaType } from 'ajv'
import { existsSync, readFileSync } from 'fs'
import path from 'path'
import fetch from 'node-fetch'
import { DEFAULT_RELEASE_VERSION, DEFAULT_CWPLUS_VERSION } from './constants'
import { AccAddress } from '@terra-money/terra.js'
import { assertions } from '@chainlink/gauntlet-core/dist/utils'

export type CONTRACT_LIST = typeof CONTRACT_LIST[keyof typeof CONTRACT_LIST]
export const CONTRACT_LIST = {
  FLAGS: 'flags',
  DEVIATION_FLAGGING_VALIDATOR: 'deviation_flagging_validator',
  OCR_2: 'ocr2',
  PROXY_OCR_2: 'proxy_ocr2',
  ACCESS_CONTROLLER: 'access_controller',
  CW20_BASE: 'cw20_base',
  MULTISIG: 'cw3_flex_multisig',
  CW4_GROUP: 'cw4_group',
} as const

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

export abstract class Contract {
  // Contract metadata, initialized in constructor
  readonly id: CONTRACT_LIST
  readonly defaultVersion: string
  readonly dirName: string
  readonly downloadUrl: string

  // Only load bytecode & schema later if needed
  version: string
  abi: TerraABI
  bytecode: string

  constructor(id, dirName, defaultVersion) {
    this.id = id
    this.defaultVersion = defaultVersion
    this.dirName = dirName
  }

  loadContractCode = async (version = this.defaultVersion): Promise<void> => {
    assertions.assert(
      !this.version || version == this.version,
      `Loading multiple versions (${this.version} and ${version}) of the same contract is unsupported.`,
    )
    this.version = version

    if (version === 'local') {
      // Possible paths depending on how/where gauntlet is being executed
      const possibleContractPaths = [
        path.join(__dirname, '../../artifacts/bin'),
        path.join(process.cwd(), './artifacts/bin'),
        path.join(process.cwd(), './tests/e2e/common_artifacts'),
        path.join(process.cwd(), './packages-ts/gauntlet-terra-contracts/artifacts/bin'),
      ]

      const codes = possibleContractPaths
        .filter((contractPath) => existsSync(`${contractPath}/${this.id}.wasm`))
        .map((contractPath) => {
          const wasm = readFileSync(`${contractPath}/${this.id}.wasm`)
          return wasm.toString('base64')
        })
      this.bytecode = codes[0]
    } else {
      const url = `${this.downloadUrl}${version}/${this.id}.wasm`
      logger.loading(`Fetching ${url}...`)
      const response = await fetch(url)
      const body = await response.arrayBuffer()
      if (body.length == 0) {
        throw new Error(`Download ${this.id}.wasm failed`)
      }
      this.bytecode = Buffer.from(body).toString('base64')
    }
  }

  loadContractABI = async (): Promise<void> => {
    // Possible paths depending on how/where gauntlet is being executed
    const cwd = process.cwd()
    const possibleContractPaths = [
      path.join(__dirname, './artifacts/contracts'),
      path.join(cwd, './contracts'),
      path.join(cwd, '../../contracts'),
      path.join(cwd, './packages-ts/gauntlet-terra-contracts/artifacts/contracts'),
      path.join(cwd, './packages-ts/gauntlet-terra-cw-plus/artifacts/contracts'),
    ]

    const abi = possibleContractPaths
      .filter((path) => existsSync(`${path}/${this.dirName}/schema`))
      .map((contractPath) => {
        const toPath = (type) => {
          if (this.id == CONTRACT_LIST.CW20_BASE && type == 'execute_msg') {
            return path.join(contractPath, `./${this.dirName}/schema/cw20_${type}`)
          } else {
            return path.join(contractPath, `./${this.dirName}/schema/${type}`)
          }
        }
        return {
          execute: io.readJSON(toPath('execute_msg')),
          query: io.readJSON(toPath('query_msg')),
          instantiate: io.readJSON(toPath('instantiate_msg')),
        }
      })
    if (abi.length === 0) {
      logger.error(`ABI not found for contract ${this.id}`)
    }

    this.abi = abi[0]
  }
}

class ChainlinkContract extends Contract {
  readonly downloadUrl = `https://github.com/smartcontractkit/chainlink-terra/releases/download/`

  constructor(id, dirName, defaultVersion = DEFAULT_RELEASE_VERSION) {
    super(id, dirName, defaultVersion)
  }
}

class CosmWasmContract extends Contract {
  readonly downloadUrl = `https://github.com/CosmWasm/cw-plus/releases/download/`

  constructor(id, dirName, defaultVersion = DEFAULT_CWPLUS_VERSION) {
    super(id, dirName, defaultVersion)
  }
}

class Contracts {
  contracts: Map<CONTRACT_LIST, Contract>

  constructor() {
    this.contracts = new Map<CONTRACT_LIST, Contract>()
  }

  // Retrieves a specific Contract object from the contract index, while loading its abi
  // and bytecode from disk or network if they haven't been already.
  async getContractWithSchemaAndCode(id: CONTRACT_LIST, version: string): Promise<Contract> {
    const contract = this.contracts[id]
    if (!contract) {
      throw new Error(`Contract ${id} not found!`)
    }
    await Promise.all([
      contract.abi ? Promise.resolve() : contract.loadContractABI(),
      contract.bytecode ? Promise.resolve() : contract.loadContractCode(version),
    ])
    return contract
  }

  addChainlink = (id: CONTRACT_LIST, dirName: string) => {
    this.contracts[id] = new ChainlinkContract(id, dirName)
    return this
  }

  addCosmwasm = (id: CONTRACT_LIST, dirName: string) => {
    this.contracts[id] = new CosmWasmContract(id, dirName)
    return this
  }
}

export const contracts = new Contracts()
  .addChainlink(CONTRACT_LIST.FLAGS, 'flags')
  .addChainlink(CONTRACT_LIST.DEVIATION_FLAGGING_VALIDATOR, 'deviation-flagging-validator')
  .addChainlink(CONTRACT_LIST.OCR_2, 'ocr2')
  .addChainlink(CONTRACT_LIST.PROXY_OCR_2, 'proxy-ocr2')
  .addChainlink(CONTRACT_LIST.ACCESS_CONTROLLER, 'access-controller')
  .addCosmwasm(CONTRACT_LIST.CW20_BASE, 'cw20_base')
  .addCosmwasm(CONTRACT_LIST.CW4_GROUP, 'cw4_group')
  .addCosmwasm(CONTRACT_LIST.MULTISIG, 'cw3_flex_multisig')
