import { bech32 } from 'bech32'
import { AccAddress, LCDClient } from '@terra-money/terra.js'

// TODO: This function should be transfered to gauntlet-core repo.
export function dateFromUnix(unixTimestamp: number): Date {
  return new Date(unixTimestamp * 1000)
}

export const getLatestContractEvent = async (provider: LCDClient, event: string, contract: AccAddress) => {
  let transmissionTx = (
    await provider.tx.search({
      events: [
        {
          key: `${event}.contract_address`,
          value: contract,
        },
      ],
      'pagination.limit': '1',
      order_by: 'ORDER_BY_DESC',
    })
  ).txs

  return transmissionTx[0]?.logs?.[0].eventsByType[event]
}
export const hexToBase64 = (s: string): string => Buffer.from(s, 'hex').toString('base64')
