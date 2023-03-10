import { Events } from '../lib/provider'

export type TransactionResponse = {
  hash: string
  address?: string
  wait: () => { success: boolean }
  tx?: any // TODO
  events?: Events
}
