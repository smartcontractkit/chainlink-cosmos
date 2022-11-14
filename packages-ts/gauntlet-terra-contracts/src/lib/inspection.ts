import { EventsByType } from '@terra-money/terra.js'
import { AccAddress, LCDClient } from '@chainlink/gauntlet-terra'
import { providerUtils } from '@chainlink/gauntlet-terra'
import { logger } from '@chainlink/gauntlet-core/dist/utils'

export type RoundData = {
  roundId: number
  answer: string
  observationsTimestamp: number
  transmissionTimestamp: number
}

// TODO: find the right place for this function
export const getLatestOCRConfigEvent = async (provider: LCDClient, contract: AccAddress) => {
  // The contract only stores the block where the config was accepted. The tx log contains the config
  const latestConfigDetails: any = await provider.wasm.contractQuery(contract, 'latest_config_details' as any)
  const setConfigTx = providerUtils.filterTxsByEvent(
    await providerUtils.getBlockTxs(provider, latestConfigDetails.block_number),
    'wasm-set_config',
  )

  return setConfigTx?.logs?.[0].eventsByType['wasm-set_config']
}

export const getLatestOCRNewTransmissionEvents = async (
  provider: LCDClient,
  contract: AccAddress,
  amount = 10,
): Promise<EventsByType[]> => {
  try {
    return providerUtils.getLatestContractEvents(provider, 'wasm-new_transmission', contract, amount)
  } catch (e) {
    logger.error(`Error fetching latest transmission events: ${e.message}`)
    return []
  }
}
