import pako from 'pako'
import { InjectiveSigningStargateClient } from '@injectivelabs/sdk-ts/dist/cjs/core/stargate'
import { logs, Coin, createProtobufRpcClient, SearchTxQuery, IndexedTx, QueryClient } from '@cosmjs/stargate'
import { sha256 } from '@cosmjs/crypto'
import { fromUtf8, toHex, toUtf8 } from '@cosmjs/encoding'
import { StdFee } from '@cosmjs/amino'
import { Uint53 } from '@cosmjs/math'
import { assert } from '@cosmjs/utils'
import { AccessConfig } from 'cosmjs-types/cosmwasm/wasm/v1/types'
import { MsgStoreCode, MsgInstantiateContract, MsgExecuteContract } from 'cosmjs-types/cosmwasm/wasm/v1/tx'
import { QueryClientImpl } from 'cosmjs-types/cosmwasm/wasm/v1/query'
import {
  ExecuteInstruction,
  ExecuteResult,
  InstantiateOptions,
  InstantiateResult,
  JsonObject,
  MsgExecuteContractEncodeObject,
  MsgInstantiateContractEncodeObject,
  MsgStoreCodeEncodeObject,
  UploadResult,
  SigningCosmWasmClientOptions,
  Contract,
} from '@cosmjs/cosmwasm-stargate'
import { SigningClient } from './client'
import Long from 'long'
import { EncodeObject, OfflineSigner } from '@cosmjs/proto-signing'
import { SigningStargateClientOptions } from '@injectivelabs/sdk-ts/dist/cjs/core/stargate/SigningStargateClient'
import { HttpEndpoint, Tendermint37Client, TendermintClient } from '@cosmjs/tendermint-rpc'

// InjectiveClient extends InjectiveSigningStargateClient to match the interface of SigningClient
// Upload, execute, instantiate methods adapted from https://github.com/cosmos/cosmjs/blob/v0.30.1/packages/cosmwasm-stargate/src/signingcosmwasmclient.ts
export class InjectiveClient extends InjectiveSigningStargateClient implements SigningClient {
  protected readonly queryService: QueryClientImpl

  constructor(tmClient: TendermintClient, signer: OfflineSigner, options?: SigningStargateClientOptions) {
    super(tmClient, signer, options)
    const queryClient = QueryClient.withExtensions(tmClient)
    const rpc = createProtobufRpcClient(queryClient)
    this.queryService = new QueryClientImpl(rpc)
    // register type urls
    this.registry.register('/cosmwasm.wasm.v1.MsgStoreCode', MsgStoreCode)
    this.registry.register('/cosmwasm.wasm.v1.MsgInstantiateContract', MsgInstantiateContract)
    this.registry.register('/cosmwasm.wasm.v1.MsgExecuteContract', MsgExecuteContract)
  }

  /**
   * Creates an instance by connecting to the given Tendermint RPC endpoint.
   *
   * For now this uses the Tendermint 0.34 client. If you need Tendermint 0.37
   * support, see `createWithSigner`.
   */
  public static async connectWithSigner(
    endpoint: string | HttpEndpoint,
    signer: OfflineSigner,
    options: SigningStargateClientOptions = {},
  ): Promise<InjectiveClient> {
    const tmClient = await Tendermint37Client.connect(endpoint)
    return InjectiveClient.createWithSigner(tmClient, signer, options)
  }

  /**
   * Creates an instance from a manually created Tendermint client.
   * Use this to use `Tendermint37Client` instead of `Tendermint34Client`.
   */
  public static async createWithSigner(
    tmClient: TendermintClient,
    signer: OfflineSigner,
    options: SigningStargateClientOptions = {},
  ): Promise<InjectiveClient> {
    return new InjectiveClient(tmClient, signer, options)
  }

  public async upload(
    senderAddress: string,
    wasmCode: Uint8Array,
    fee: StdFee | 'auto' | number,
    memo = '',
  ): Promise<UploadResult> {
    const compressed = pako.gzip(wasmCode, { level: 9 })
    const storeCodeMsg: MsgStoreCodeEncodeObject = {
      typeUrl: '/cosmwasm.wasm.v1.MsgStoreCode',
      value: MsgStoreCode.fromPartial({
        sender: senderAddress,
        wasmByteCode: compressed,
      }),
    }

    const result = await this.signAndBroadcast(senderAddress, [storeCodeMsg], fee, memo)
    if (!!result.code) {
      throw new Error(
        `Error when broadcasting tx ${result.transactionHash} at height ${result.height}. Code: ${result.code}; Raw log: ${result.rawLog}`,
      )
    }
    const parsedLogs = logs.parseRawLog(result.rawLog)
    const codeIdAttr = logs.findAttribute(parsedLogs, 'cosmwasm.wasm.v1.EventCodeStored', 'code_id')
    return {
      originalSize: wasmCode.length,
      originalChecksum: toHex(sha256(wasmCode)),
      compressedSize: compressed.length,
      compressedChecksum: toHex(sha256(compressed)),
      codeId: Number.parseInt(JSON.parse(codeIdAttr.value), 10),
      logs: parsedLogs,
      height: result.height,
      transactionHash: result.transactionHash,
      events: result.events,
      gasWanted: result.gasWanted,
      gasUsed: result.gasUsed,
    }
  }

  public async instantiate(
    senderAddress: string,
    codeId: number,
    msg: JsonObject,
    label: string,
    fee: StdFee | 'auto' | number,
    options: InstantiateOptions = {},
  ): Promise<InstantiateResult> {
    const instantiateContractMsg: MsgInstantiateContractEncodeObject = {
      typeUrl: '/cosmwasm.wasm.v1.MsgInstantiateContract',
      value: MsgInstantiateContract.fromPartial({
        sender: senderAddress,
        codeId: Long.fromString(new Uint53(codeId).toString()),
        label: label,
        msg: toUtf8(JSON.stringify(msg)),
        funds: [...(options.funds || [])],
        admin: options.admin,
      }),
    }
    const result = await this.signAndBroadcast(senderAddress, [instantiateContractMsg], fee, options.memo)
    if (!!result.code) {
      throw new Error(
        `Error when broadcasting tx ${result.transactionHash} at height ${result.height}. Code: ${result.code}; Raw log: ${result.rawLog}`,
      )
    }
    const parsedLogs = logs.parseRawLog(result.rawLog)
    const contractAddressAttr = logs.findAttribute(
      parsedLogs,
      'cosmwasm.wasm.v1.EventContractInstantiated',
      'contract_address',
    )
    return {
      contractAddress: contractAddressAttr.value,
      logs: parsedLogs,
      height: result.height,
      transactionHash: result.transactionHash,
      events: result.events,
      gasWanted: result.gasWanted,
      gasUsed: result.gasUsed,
    }
  }

  public async execute(
    senderAddress: string,
    contractAddress: string,
    msg: JsonObject,
    fee: StdFee | 'auto' | number,
    memo = '',
    funds?: readonly Coin[],
  ): Promise<ExecuteResult> {
    const instruction: ExecuteInstruction = {
      contractAddress: contractAddress,
      msg: msg,
      funds: funds,
    }
    return this.executeMultiple(senderAddress, [instruction], fee, memo)
  }

  /**
   * Like `execute` but allows executing multiple messages in one transaction.
   */
  public async executeMultiple(
    senderAddress: string,
    instructions: readonly ExecuteInstruction[],
    fee: StdFee | 'auto' | number,
    memo = '',
  ): Promise<ExecuteResult> {
    const msgs: MsgExecuteContractEncodeObject[] = instructions.map((i) => ({
      typeUrl: '/cosmwasm.wasm.v1.MsgExecuteContract',
      value: MsgExecuteContract.fromPartial({
        sender: senderAddress,
        contract: i.contractAddress,
        msg: toUtf8(JSON.stringify(i.msg)),
        funds: [...(i.funds || [])],
      }),
    }))
    const result = await this.signAndBroadcast(senderAddress, msgs, fee, memo)
    if (!!result.code) {
      throw new Error(
        `Error when broadcasting tx ${result.transactionHash} at height ${result.height}. Code: ${result.code}; Raw log: ${result.rawLog}`,
      )
    }
    return {
      logs: logs.parseRawLog(result.rawLog),
      height: result.height,
      transactionHash: result.transactionHash,
      events: result.events,
      gasWanted: result.gasWanted,
      gasUsed: result.gasUsed,
    }
  }

  /**
   * Throws an error if no contract was found at the address
   */
  public async getContract(address: string): Promise<Contract> {
    const { address: retrievedAddress, contractInfo } = await this.queryService.ContractInfo({ address })
    if (!contractInfo) throw new Error(`No contract found at address "${address}"`)
    assert(retrievedAddress, 'address missing')
    assert(contractInfo.codeId && contractInfo.creator && contractInfo.label, 'contractInfo incomplete')
    return {
      address: retrievedAddress,
      codeId: contractInfo.codeId.toNumber(),
      creator: contractInfo.creator,
      admin: contractInfo.admin || undefined,
      label: contractInfo.label,
      ibcPortId: contractInfo.ibcPortId || undefined,
    }
  }

  /**
   * Makes a smart query on the contract, returns the parsed JSON document.
   *
   * Promise is rejected when contract does not exist.
   * Promise is rejected for invalid query format.
   * Promise is rejected for invalid response format.
   */
  public async queryContractSmart(address: string, queryMsg: JsonObject): Promise<JsonObject> {
    let data: Uint8Array
    try {
      const request = { address: address, queryData: toUtf8(JSON.stringify(queryMsg)) }
      ;({ data } = await this.queryService.SmartContractState(request))
    } catch (error) {
      if (error.message.startsWith('not found: contract')) {
        throw new Error(`No contract found at address "${address}"`)
      } else {
        throw error
      }
    }
    // By convention, smart queries must return a valid JSON document (see https://github.com/CosmWasm/cosmwasm/issues/144)
    let responseText: string
    try {
      responseText = fromUtf8(data)
    } catch (error) {
      throw new Error(`Could not UTF-8 decode smart query response from contract: ${error}`)
    }
    try {
      return JSON.parse(responseText)
    } catch (error) {
      throw new Error(`Could not JSON parse smart query response from contract: ${error}`)
    }
  }
}
