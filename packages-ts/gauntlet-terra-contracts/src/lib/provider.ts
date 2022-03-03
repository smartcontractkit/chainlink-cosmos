import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { TxInfo } from '@terra-money/terra.js'
import { Search } from '../commands/abstract/inspectionWrapper'

export const filterTxsByEvent = (txs: TxInfo[], event: string): TxInfo | undefined => {
  const filteredTx = txs.filter((tx) => tx.logs?.some((log) => log.eventsByType[event]))?.[0]
  return filteredTx
}

export const getBlockTxs = async (search: Search, block: number, offset = 0): Promise<TxInfo[]> => {
  // recursive call to get every tx in the block. API has a 100 tx limit. Increasing the offset 100 every time
  try {
    const txs = await search({
      events: [
        {
          key: 'tx.height',
          value: String(block),
        },
      ],
      'pagination.offset': String(offset),
    })
    return txs.txs.concat(await getBlockTxs(search, block, offset + 100))
  } catch (e) {
    logger.debug('No more txs in block')
  }

  return []
}
