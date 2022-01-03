export type TransactionResponse = {
  hash: string
  address?: string
  wait: () => { success: boolean }
  tx?: any
}
