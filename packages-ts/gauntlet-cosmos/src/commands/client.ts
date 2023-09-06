import {
  Contract,
  ExecuteResult,
  InstantiateOptions,
  InstantiateResult,
  JsonObject,
  UploadResult,
} from '@cosmjs/cosmwasm-stargate'
import { AccessConfig } from 'cosmjs-types/cosmwasm/wasm/v1/types'
import { Coin, EncodeObject } from '@cosmjs/proto-signing'
import { DeliverTxResponse, IndexedTx, SearchTxQuery, StdFee } from '@cosmjs/stargate'

export interface SigningClient {
  signAndBroadcast(
    signerAddress: string,
    messages: readonly EncodeObject[],
    fee: StdFee | 'auto' | number,
    memo?: string,
  ): Promise<DeliverTxResponse>
  execute(
    senderAddress: string,
    contractAddress: string,
    msg: JsonObject,
    fee: StdFee | 'auto' | number,
    memo?: string,
    funds?: readonly Coin[],
  ): Promise<ExecuteResult>
  instantiate(
    senderAddress: string,
    codeId: number,
    msg: JsonObject,
    label: string,
    fee: StdFee | 'auto' | number,
    options?: InstantiateOptions,
  ): Promise<InstantiateResult>
  upload(
    senderAddress: string,
    wasmCode: Uint8Array,
    fee: StdFee | 'auto' | number,
    memo?: string,
    instantiatePermission?: AccessConfig,
  ): Promise<UploadResult>
  simulate(signerAddress: string, messages: readonly EncodeObject[], memo: string | undefined): Promise<number>
  searchTx(query: SearchTxQuery): Promise<readonly IndexedTx[]>
  getBalance(address: string, searchDenom: string): Promise<Coin>
  getChainId(): Promise<string>
  getContract(address: string): Promise<Contract>
  queryContractSmart(address: string, queryMsg: JsonObject): Promise<JsonObject>
}
