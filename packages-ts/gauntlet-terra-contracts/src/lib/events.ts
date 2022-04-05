import { AccAddress } from '@terra-money/terra.js'

type EventOraclePaid = {
  transmitter: string
  payee: string
  amount: string
  linkToken: string
  method?: string
}

export const parseOraclePaidEvent = (event): EventOraclePaid => {
  // Parse and validate every value
  const transmitter = event.transmitter[0]
  const payee = event.payee[0]
  const amount = event.amount[0]
  const link_token = event.link_token[0]

  if (!AccAddress.validate(transmitter)) throw new Error(`Invalid transmitter address`)
  if (!AccAddress.validate(payee)) throw new Error(`Invalid payee wallet address`)
  if (!AccAddress.validate(link_token)) throw new Error(`Invalid LINK token contract address`)

  return {
    transmitter: transmitter,
    payee: payee,
    amount: amount,
    linkToken: link_token,
  }
}
