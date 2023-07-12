import { AccAddress, Client, providerUtils } from '@chainlink/gauntlet-cosmos'
import { logger } from '@chainlink/gauntlet-core/dist/utils'

export type RoundData = {
  roundId: number
  answer: string
  observationsTimestamp: number
  transmissionTimestamp: number
}

// TODO: maybe use blockSearchAll via the tendermint client

// TODO: find the right place for this function
export const getLatestOCRConfigEvent = async (provider: Client, contract: AccAddress) => {
  // The contract only stores the block where the config was accepted. The tx log contains the config
  const latestConfigDetails: any = await provider.queryContractSmart(contract, 'latest_config_details' as any)
  const setConfigTx = providerUtils.filterTxsByEvent(
    // TODO: there has to be a way to filter by tag for event then scan single block
    await provider.searchTx({ height: latestConfigDetails.block_number }),
    'wasm-set_config',
  )

  return setConfigTx?.events?.[0]
}

export const getLatestOCRNewTransmissionEvents = async (
  provider: Client,
  contract: AccAddress,
): Promise<providerUtils.Events> => {
  try {
    return providerUtils.getLatestContractEvents(provider, 'wasm-new_transmission', contract)
  } catch (e) {
    logger.error(`Error fetching latest transmission events: ${e.message}`)
    return []
  }
}
