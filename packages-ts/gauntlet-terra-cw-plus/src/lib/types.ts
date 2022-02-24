import { AccAddress } from '@terra-money/terra.js'

export enum Vote {
  YES = 'yes',
  NO = 'no',
  ABS = 'abstain',
  VETO = 'veto',
}

export enum Action {
  CREATE = 'create',
  APPROVE = 'approve',
  EXECUTE = 'execute',
  NONE = 'none',
}

export type WasmMsg = {
  wasm: {
    execute: {
      contract_addr: string
      funds: {
        denom: string
        amount: string
      }[]
      msg: string
    }
  }
}

export type State = {
  threshold: number
  nextAction: Action
  owners: AccAddress[]
  approvers: string[]
  // https://github.com/CosmWasm/cw-plus/blob/82138f9484e538913f7faf78bc292fb14407aae8/packages/cw3/src/query.rs#L75
  currentStatus?: 'pending' | 'open' | 'rejected' | 'passed' | 'executed'
  data?: WasmMsg[]
  expiresAt?: Date
}
