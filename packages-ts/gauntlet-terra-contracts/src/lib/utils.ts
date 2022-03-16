import { bech32 } from 'bech32'
import { CONTRACT_LIST, contracts } from './contracts'
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

enum MODE {
  PLAIN,
  COLORIZED,
  DEBUG,
}

// fmtAddress:  Automatically format terra addresses depending on contract type.
//
// Use ${fmtAddress(address)} instead of ${address} in strings sent to console or log.
//  - If it matches the multisig address read from environment, the address will show up
//    as brown and labelled "multisig".
//  - If it matches a known contract address read from the environemnt (LINK, BILLING_ACCESS_CONTROLLER,... ),
//    the address will be blue and labelled with the contract name.
//  - Unknown addresses will show up as yellow.
export const fmtAddress = (address: AccAddress, mode = MODE.DEBUG): string => {
  const modePrefix = {
    [MODE.COLORIZED]: '[',
    [MODE.PLAIN]: '',
    [MODE.DEBUG]: '[', // show generated ansi codes
  }

  const defColor = (color) => (mode == MODE.PLAIN ? '' : color)
  const esc = modePrefix[mode]

  const brown = defColor(`${esc}2;31m`)
  const dimYellow = defColor(`${esc}2;33m`)
  const blue = defColor(`${esc}0;34m`)
  const dimBlue = defColor(`${esc}2;34m`)
  const yellow = defColor(`${esc}0;33m`)
  const reset = defColor(`${esc}0;0m`)

  const contractIds = Object.values(CONTRACT_LIST)
  const contractAddresses = contractIds
    .map((id) => contracts[id].addresses)
    .filter((aList) => aList.length > 0)
    .join()
    .split(',')

  if (address == contracts[CONTRACT_LIST.MULTISIG].address) {
    return `[${brown}multisig🧳${dimYellow}${address}${reset}]`
  } else if (contractAddresses.includes(address)) {
    const id = contractIds.filter((id) => contracts[id].address == address)[0] as CONTRACT_LIST
    return `[${blue}${id}📜${dimBlue}${address} ${reset}]`
  } else {
    return `[${yellow}👝${address} ${reset}]`
  }
}
