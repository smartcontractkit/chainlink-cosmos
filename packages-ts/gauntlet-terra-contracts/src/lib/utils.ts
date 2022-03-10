import { bech32 } from 'bech32'
import { multisig } from '@chainlink/gauntlet-terra-cw-plus'
import { AccAddress } from '@terra-money/terra.js'

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

export const fmtAddress = (address: AccAddress): string => {
  const e = '\033['
  return address == multisig
    ? `[${e}2;31m multisig🧳${e}2;33m${multisig} ${e}0;0m]`
    : `[${e}0;33m👝${address} ${e}0;0m]`
}
