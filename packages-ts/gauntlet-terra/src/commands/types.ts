import { BlockTxBroadcastResult, EventsByType } from '@terra-money/terra.js'

export type TransactionResponse = {
  hash: string
  address?: string
  wait: () => { success: boolean }
  tx?: BlockTxBroadcastResult
  events?: EventsByType[]
}

// TODO:  move to gauntlet-core
export type ContractId = string
