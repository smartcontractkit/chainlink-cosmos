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
    execute?: {
      contract_addr: string
      funds: Coin[]
      msg: string
    }
    update_admin?: {
      admin: string
      contract_addr: string
    }
    migrate?: {
      contract_addr: string
      msg: string
      new_code_id: number
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
  multisig: {
    threshold: number
    owners: AccAddress[]
    maxVotingPeriod: number
  }
  proposal: {
    id?: number
    nextAction: Action
    currentStatus?: 'pending' | 'open' | 'rejected' | 'passed' | 'executed'
    data?: Cw3WasmMsg[]
    expiresAt?: Date
    approvers: string[]
  }
}

export type Expiration = {
  at_time: string // Uint64
}
