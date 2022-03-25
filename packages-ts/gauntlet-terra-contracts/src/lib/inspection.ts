import { AccAddress, LCDClient } from '@terra-money/terra.js'
import { providerUtils } from '@chainlink/gauntlet-terra'

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
