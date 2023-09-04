import { Result, WriteCommand, AddressBook } from '@chainlink/gauntlet-core'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { withProvider, withCodeIds, withNetwork } from '../middlewares'
import { toUtf8 } from '@cosmjs/encoding'
import {
  ExecuteResult,
  InstantiateResult,
  MsgExecuteContractEncodeObject,
  UploadResult,
} from '@cosmjs/cosmwasm-stargate'
import { MsgExecuteContract } from 'cosmjs-types/cosmwasm/wasm/v1/tx'
import { TransactionResponse } from '../types'
import { AccountData, EncodeObject, OfflineSigner } from '@cosmjs/proto-signing'
import { AccAddress } from '../..'
import { assertIsDeliverTxSuccess } from '@cosmjs/stargate'
import { SigningClient } from '../client'

type CodeIds = Record<string, number>

export default abstract class CosmosCommand extends WriteCommand<TransactionResponse> {
  provider: SigningClient
  wallet: OfflineSigner
  signer: AccountData
  addressBook: AddressBook
  contracts: string[]
  public codeIds: CodeIds

  abstract execute: () => Promise<Result<TransactionResponse>>
  abstract makeRawTransaction: (signer: AccAddress) => Promise<EncodeObject[]>
  // Preferable option to initialize the command instead of new CosmosCommand. This should be an static option to construct the command
  buildCommand?: (flags, args) => Promise<CosmosCommand>
  beforeExecute: (context?: any) => Promise<void>

  afterExecute = async (response: Result<TransactionResponse>): Promise<any> => {
    logger.success(`Execution finished at transaction: ${response.responses[0].tx.hash}`)
  }

  constructor(flags, args) {
    super(flags, args)
    this.use(withNetwork, withProvider, withCodeIds)
  }

  parseResponseValue(receipt: any, eventType: string, attributeType: string) {
    try {
      const parsed = JSON.parse(receipt?.raw_log)
      const event = parsed[0].events.filter((event) => event.type === eventType)[0]
      if (event) {
        const attribute = event.attributes.filter((attr) => attr.key === attributeType)[0]
        return attribute.value
      }
    } catch (e) {
      if (receipt?.raw_log.includes('insufficient funds')) {
        throw Error(`Wallet does not have enough funds for txn: ${receipt?.raw_log}`)
      } else {
        logger.log('Error parsing response', e.message)
        return undefined
      }
    }
  }

  // TODO: need to add type of tx, address is parsed only for intantiation
  wrapResponse = (tx: any): TransactionResponse => ({
    hash: tx.txhash,
    address: this.parseResponseValue(tx, 'instantiate_contract', 'contract_address'), // TODO: handle insufficient funds logs
    wait: () => ({
      success: tx.events.length > 0 && !tx.code,
    }),
    tx,
    events: tx.events,
  })

  signAndSend = async (messages: EncodeObject[]): Promise<TransactionResponse> => {
    let senderAddress = this.signer.address
    logger.loading('Signing and sending transaction...')
    const result = await this.provider.signAndBroadcast(senderAddress, messages, 'auto')
    assertIsDeliverTxSuccess(result)
    return
    // return this.wrapResponse(result)
  }

  // "call" is execute
  async call(contractAddress: string, msg: any): Promise<ExecuteResult> {
    let senderAddress = (await this.wallet.getAccounts())[0].address

    const result = await this.provider.execute(senderAddress, contractAddress, msg, 'auto')

    return result
  }

  /// TODO: rename this.signer into sender

  // "deploy" is instantiate
  async deploy(codeId: number, msg: any): Promise<InstantiateResult> {
    let senderAddress = this.signer.address
    let label = 'Label'

    logger.loading(`Deploying contract...`)
    const result = await this.provider.instantiate(senderAddress, codeId, msg, label, 'auto', {
      memo: 'Instantiating',
      admin: senderAddress,
    })

    return result
  }

  async upload(wasmCode: Uint8Array, contractName: string): Promise<UploadResult> {
    let senderAddress = this.signer.address
    let memo = `Storing ${contractName}`
    logger.loading(`Uploading ${contractName} contract code...`)
    const response = await this.provider.upload(senderAddress, wasmCode, 'auto', memo)
    // TODO: custom wrapResponse for upload
    return response
  }

  // TODO: replace simulate with relying on `gas: "auto"` on transmit?
  async simulate(signer: AccAddress, msgs: EncodeObject[]): Promise<Number> {
    // gas estimation successful => tx is valid (simulation is run under the hood)
    try {
      return await this.provider.simulate(signer, msgs, '')
    } catch (e) {
      // TODO: parse message
      // const message = e.response?.data?.message || e.message || e
      throw new Error(`Simulation Failed: ${e}`)
    }
  }
  // TODO: accept a cosmjs-types Msg type
  async simulateExecute(contractAddress: string, inputs: any[]): Promise<Number> {
    const signer = this.signer.address

    const msgs = inputs.map((input) => {
      const msg: MsgExecuteContractEncodeObject = {
        typeUrl: '/cosmwasm.wasm.v1.MsgExecuteContract',
        value: MsgExecuteContract.fromPartial({
          sender: signer,
          contract: contractAddress,
          msg: toUtf8(JSON.stringify(input)),
          funds: [],
        }),
      }
      return msg
    })

    return await this.simulate(signer, msgs)
  }
}
