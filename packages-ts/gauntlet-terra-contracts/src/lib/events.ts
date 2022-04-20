import { AccAddress } from '@terra-money/terra.js'

type EventOraclePaid = {
  payee: string
  amount: string
}

export const parseOraclePaidEvent = (event): EventOraclePaid | null => {
  // Parse and validate every value
  const payee = event.to[0]
  const amount = event.amount[0]

  if (!AccAddress.validate(payee)) return null

  return {
    payee: payee,
    amount: amount,
  }
}
