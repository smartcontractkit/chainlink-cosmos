import { AccAddress, LCDClient } from '@terra-money/terra.js'
import { providerUtils } from '@chainlink/gauntlet-terra'

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

export const getLatestOCRNewTransmissionEvent = async (provider: LCDClient, contract: AccAddress) => {
  let transmissionTx = (
    await provider.tx.search({
      events: [
        {
          key: 'wasm-new_transmission.contract_address',
          value: contract,
        },
      ],
    })
  ).txs

  return transmissionTx.length > 0
    ? transmissionTx[transmissionTx.length - 1]?.logs?.[0].eventsByType['wasm-new_transmission']
    : null
}

export const parseObserversByLength = (observers: string, observersNumber: number): number[] =>
  (observers.substring(0, observersNumber * 2).match(/.{2}/g) || []).map((s) => parseInt(s, 16))
