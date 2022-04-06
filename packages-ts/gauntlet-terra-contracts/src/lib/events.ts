import { AccAddress } from '@terra-money/terra.js'

type EventOraclePaid = {
  transmitter: string
  payee: string
  amount: string
  linkToken: string
  method?: string
}

export const parseOraclePaidEvent = (event): EventOraclePaid | null => {
  // Parse and validate every value
  const transmitter = event.transmitter[0]
  const payee = event.payee[0]
  const amount = event.amount[0]
  const link_token = event.link_token[0]

  if (!AccAddress.validate(transmitter)) return null
  if (!AccAddress.validate(payee)) return null
  if (!AccAddress.validate(link_token)) return null

  return {
    transmitter: transmitter,
    payee: payee,
    amount: amount,
    linkToken: link_token,
  }
}
