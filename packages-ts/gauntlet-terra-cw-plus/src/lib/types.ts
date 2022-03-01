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
  multisig: {
    threshold: number
    owners: AccAddress[]
  }
  proposal: {
    id?: number
    nextAction: Action
    currentStatus?: 'pending' | 'open' | 'rejected' | 'passed' | 'executed'
    data?: WasmMsg[]
    expiresAt?: Date
    approvers: string[]
  }
}
