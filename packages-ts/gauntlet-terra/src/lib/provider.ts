import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { AccAddress, EventsByType, LCDClient, TxInfo } from '@terra-money/terra.js'

export const filterTxsByEvent = (txs: TxInfo[], event: string): TxInfo | undefined => {
  const filteredTx = txs.filter((tx) => tx.logs?.some((log) => log.eventsByType[event]))?.[0]
  return filteredTx
}

export const getBlockTxs = async (provider: LCDClient, block: number, offset = 0): Promise<TxInfo[]> => {
  // recursive call to get every tx in the block. API has a 100 tx limit. Increasing the offset 100 every time
  try {
    const txs = await provider.tx.search({
      events: [
        {
          key: 'tx.height',
          value: String(block),
        },
      ],
      'pagination.offset': String(offset),
    })
    return txs.txs.concat(await getBlockTxs(provider, block, offset + 100))
  } catch (e) {
    const expectedError = 'page should be within'
    if (!((e.response?.data?.message as string) || '').includes(expectedError)) {
      logger.error(`Error fetching block ${block} and offset ${offset}: ${e.response?.data?.message || e.message}`)
      return []
    }
    logger.debug(`No more txs in block ${block}. Last offset ${offset}`)
  }

  return []
}

export const getLatestContractEvents = async (
  provider: LCDClient,
  event: string,
  contract: AccAddress,
  paginationLimit = 1,
): Promise<EventsByType[]> => {
  const txs: TxInfo[] = (
    await provider.tx.search({
      events: [
        {
          key: `${event}.contract_address`,
          value: contract,
        },
      ],
      'pagination.limit': String(paginationLimit),
      order_by: 'ORDER_BY_DESC',
    })
  ).txs

  if (txs.length === 0) return []
  const events = txs
    .map(({ logs }) => logs?.map(({ eventsByType }) => eventsByType) || [])
    .reduce((acc, eventsByType) => [...acc, ...eventsByType], [])

  return events
}

export const getBalance = async (provider: LCDClient, address: AccAddress): Promise<string | null> => {
  try {
    return (await provider.bank.balance(address))[0].toString()
  } catch (e) {
    logger.error(`Error fetching ${address} balance: ${e.message}`)
    return null
  }
}
