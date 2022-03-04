import { Proto } from '@chainlink/gauntlet-core/dist/crypto'
import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { AccAddress, LCDClient } from '@terra-money/terra.js'
import { providerUtils } from '@chainlink/gauntlet-terra'

export const toComparableNumber = (v: string | number) => new BN(v).toString()
export const toComparableLongNumber = (v: Long) => new BN(Proto.Protobuf.longToString(v)).toString()
export const wrappedComparableLongNumber = (v: any) => {
  // Proto encoding will ignore falsy values.
  if (!v) return '0'
  return toComparableLongNumber(v)
}

export const getLatestOCRConfig = async (provider: LCDClient, contract: AccAddress) => {
  const latestConfigDetails: any = await provider.wasm.contractQuery(contract, 'latest_config_details' as any)

  const setConfigTx = providerUtils.filterTxsByEvent(
    await providerUtils.getBlockTxs(provider, latestConfigDetails.block_number),
    'wasm-set_config',
  )

  return setConfigTx?.logs?.[0].eventsByType['wasm-set_config']
}
