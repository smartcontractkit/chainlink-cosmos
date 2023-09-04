import pako from 'pako'
import { InjectiveSigningStargateClient } from '@injectivelabs/sdk-ts/dist/cjs/core/stargate'
import { logs, Coin, DeliverTxResponse } from '@cosmjs/stargate'
import { sha256 } from '@cosmjs/crypto'
import { toHex, toUtf8 } from '@cosmjs/encoding'
import { StdFee } from '@cosmjs/amino'
import { Uint53 } from '@cosmjs/math'
import { AccessConfig } from 'cosmjs-types/cosmwasm/wasm/v1/types'
import { MsgStoreCode, MsgInstantiateContract, MsgExecuteContract } from 'cosmjs-types/cosmwasm/wasm/v1/tx'
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
} from '@cosmjs/cosmwasm-stargate'
import { SigningClient } from './client'
import Long from 'long'
import { EncodeObject, OfflineSigner } from '@cosmjs/proto-signing'
import { SigningStargateClientOptions } from '@injectivelabs/sdk-ts/dist/cjs/core/stargate/SigningStargateClient'
import { HttpEndpoint, Tendermint37Client, TendermintClient } from '@cosmjs/tendermint-rpc'

// InjectiveClient extends InjectiveSigningStargateClient to match the interface of SigningClient
// Upload, execute, instantiate methods adapted from https://github.com/cosmos/cosmjs/blob/a242608408b6362c16860e44c130a35d2ec09c5e/packages/cosmwasm-stargate/src/signingcosmwasmclient.ts
export class InjectiveClient extends InjectiveSigningStargateClient implements SigningClient {
  public static async connectWithSigner(
    endpoint: string | HttpEndpoint,
    signer: OfflineSigner,
    options?: SigningStargateClientOptions,
  ): Promise<InjectiveClient> {
    const tmClient: TendermintClient = await Tendermint37Client.connect(endpoint)
    return new InjectiveClient(tmClient, signer, options)
  }

  public async upload(
    senderAddress: string,
    wasmCode: Uint8Array,
    fee: StdFee | 'auto' | number,
    memo = '',
    instantiatePermission?: AccessConfig,
  ): Promise<UploadResult> {
    const compressed = pako.gzip(wasmCode, { level: 9 })
    const storeCodeMsg: MsgStoreCodeEncodeObject = {
      typeUrl: '/cosmwasm.wasm.v1.MsgStoreCode',
      value: MsgStoreCode.fromPartial({
        sender: senderAddress,
        wasmByteCode: compressed,
        instantiatePermission,
      }),
    }

    // When uploading a contract, the simulation is only 1-2% away from the actual gas usage.
    // So we have a smaller default gas multiplier than signAndBroadcast.
    const usedFee = fee == 'auto' ? 1.1 : fee

    const result = await this.signAndBroadcast(senderAddress, [storeCodeMsg], usedFee, memo)
    if (!!result.code) {
      throw new Error(
        `Error when broadcasting tx ${result.transactionHash} at height ${result.height}. Code: ${result.code}; Raw log: ${result.rawLog}`,
      )
    }
    const parsedLogs = logs.parseRawLog(result.rawLog)
    const codeIdAttr = logs.findAttribute(parsedLogs, 'store_code', 'code_id')
    return {
      checksum: toHex(sha256(wasmCode)),
      originalSize: wasmCode.length,
      compressedSize: compressed.length,
      codeId: Number.parseInt(codeIdAttr.value, 10),
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
    const contractAddressAttr = logs.findAttribute(parsedLogs, 'instantiate', '_contract_address')
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

  public async signAndBroadcast(
    signerAddress: string,
    messages: readonly EncodeObject[],
    fee: StdFee | 'auto' | number,
    memo?: string,
  ): Promise<DeliverTxResponse> {
    const result = await super.signAndBroadcast(signerAddress, messages, fee, memo)
    return {
      ...result,
      msgResponses: [], // unused, but required in cosmjs type and missing in injective type
    }
  }
}
