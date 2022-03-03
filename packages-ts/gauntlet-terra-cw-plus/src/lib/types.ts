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

type Coin = {
  denom: string
  amount: string
}

export type Cw3WasmMsg = {
  wasm: {
    execute: {
      contract_addr: string
      funds: Coin[]
      msg: string
    }
  }
}

export type Cw3BankMsg = {
  bank: {
    send: {
      amount: Coin[]
      to_address: string
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
  data?: Cw3WasmMsg[]
  expiresAt?: Date
}
