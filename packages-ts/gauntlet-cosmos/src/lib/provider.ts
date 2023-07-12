import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { AccAddress, Client } from '../index'
import { Event, IndexedTx } from '@cosmjs/cosmwasm-stargate'

export const filterTxsByEvent = (txs: readonly IndexedTx[], type: string): IndexedTx | undefined => {
  const filteredTxs = txs.filter((tx) => tx.events.some((event) => event.type === type))
  return filteredTxs?.[0]
}

export type Events = { [type: string]: any[] }[]

export const toEvent = (event: Event): any => {
  event.attributes.reduce((acc, attr) => {
    if (acc[attr.key] === undefined) {
      acc[attr.key] = attr.value
    } else {
      let array = Array.isArray(acc[attr.key]) ? acc[attr.key] : [acc[attr.key]]
      acc[attr.key] = array.push(attr.value)
    }
    return acc
  }, {})
}

export const getLatestContractEvents = async (
  provider: Client,
  event: string,
  contract: AccAddress,
): Promise<Events> => {
  let txs = await provider.searchTx({
    tags: [
      {
        key: `${event}.contract_address`,
        value: contract,
      },
    ],
  }) // TODO: ORDER_BY_DESC

  if (txs.length === 0) return []
  const events = txs.map(({ events }) =>
    events.reduce((acc, event) => {
      ;(acc[event.type] = acc[event.type] || []).push(toEvent(event))
      return acc
    }, {}),
  )

  return events
}

export const getBalance = async (provider: Client, address: AccAddress, denom: string): Promise<string | null> => {
  try {
    return (await provider.getBalance(address, denom)).amount.toString()
  } catch (e) {
    logger.error(`Error fetching ${address} balance: ${e.message}`)
    return null
  }
}
