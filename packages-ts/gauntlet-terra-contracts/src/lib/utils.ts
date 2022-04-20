import { bech32 } from 'bech32'
import { AccAddress, LCDClient } from '@terra-money/terra.js'

// https://docs.terra.money/docs/develop/sdks/terra-js/common-examples.html
export function isValidAddress(address) {
  try {
    const { prefix: decodedPrefix } = bech32.decode(address) // throw error if checksum is invalid
    // verify address prefix
    return decodedPrefix === 'terra'
  } catch {
    // invalid checksum
    return false
  }
}

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
