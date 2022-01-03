import { Result, WriteCommand } from '@chainlink/gauntlet-core'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { MsgStoreCode } from '@terra-money/terra.js'

import { withProvider, withWallet, withCodeIds, withNetwork } from '../middlewares'
import {
  BlockTxBroadcastResult,
  LCDClient,
  MsgExecuteContract,
  MsgInstantiateContract,
  TxError,
  Wallet,
} from '@terra-money/terra.js'
import { TransactionResponse } from '../types'

type CodeIds = Record<string, number>

export default abstract class TerraCommand extends WriteCommand<TransactionResponse> {
  wallet: Wallet
  provider: LCDClient
  contracts: string[]
  public codeIds: CodeIds
  abstract execute: () => Promise<Result<TransactionResponse>>

  constructor(flags, args) {
    super(flags, args)
    this.use(withNetwork, withProvider, withWallet, withCodeIds)
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
      logger.log('Error parsing response', e.message)
      return undefined
    }
  }

  // TODO: need to add type of tx, address is parsed only for intantiation
  wrapResponse = (tx: BlockTxBroadcastResult): TransactionResponse => ({
    hash: tx.txhash,
    address: this.parseResponseValue(tx, 'instantiate_contract', 'contract_address'),
    wait: () => ({
      success: tx.logs.length > 0 && !(tx as TxError)?.code,
    }),
    tx,
  })

  async query(address, input): Promise<any> {
    return await this.provider.wasm.contractQuery(address, input)
  }

  async call(address, input) {
    const msg = new MsgExecuteContract(this.wallet.key.accAddress, address, input)

    const tx = await this.wallet.createAndSignTx({
      msgs: [msg],
    })

    const res = await this.provider.tx.broadcast(tx)

    logger.debug(res)
    return this.wrapResponse(res)
  }

  async deploy(codeId, instantiateMsg, migrationContract = undefined) {
    const instantiate = new MsgInstantiateContract(
      this.wallet.key.accAddress,
      migrationContract,
      codeId,
      instantiateMsg,
    )
    const instantiateTx = await this.wallet.createAndSignTx({
      msgs: [instantiate],
      memo: 'Instantiating',
    })
    logger.loading(`Deploying contract...`)
    const res = await this.provider.tx.broadcast(instantiateTx)

    return this.wrapResponse(res)
  }

  async upload(wasm, contractName) {
    const code = new MsgStoreCode(this.wallet.key.accAddress, wasm)

    const tx = await this.wallet.createAndSignTx({
      msgs: [code],
      memo: `Storing ${contractName}`,
    })

    logger.loading(`Uploading ${contractName} contract code...`)
    const res = await this.provider.tx.broadcast(tx)

    return this.wrapResponse(res)
  }
}
