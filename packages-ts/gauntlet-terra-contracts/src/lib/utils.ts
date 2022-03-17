import { bech32 } from 'bech32'
import { CONTRACT_LIST, addressBook } from './contracts'
import { AccAddress } from '@terra-money/terra.js'
import logger from '@chainlink/gauntlet-core/dist/utils/logger'
import { TerraCommand } from '@chainlink/gauntlet-terra'
import { assert } from '@chainlink/gauntlet-core/dist/utils/assertions'
import { utils } from '@chainlink/gauntlet-terra-cw-plus/dist/commands/inspect'

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
// Note:
//  use(withAddressBook) middleware must be enabled for any commands calling this
//
// Then
//  Use ${fmtAddress(address)} instead of ${address} in strings sent to console or log.
//   - If it matches the multisig address read from environment, the address will show up
//     as brown and labelled "multisig".
//   - If it matches a known contract address read from the environemnt (LINK, BILLING_ACCESS_CONTROLLER,... ),
//     the address will be blue and labelled with the contract name.
//   - Unknown addresses will show up as yellow.
export const fmtAddress = (address: AccAddress, mode = MODE.COLORIZED): string => {
  assert(!!addressBook.operator, `fmtAddress called on Command without "use withAddressBook"`)

  const modePrefix = {
    [MODE.COLORIZED]: '[',
    [MODE.PLAIN]: '',
  }

  type COLOR = 'red' | 'blue' | 'yellow' | 'green'
  type INTENSITY = 'dim' | 'bright'
  type Style = COLOR | INTENSITY
  type Styles = {
    [key: string]: Style[]
  }
  const styles = {
    MULTISIG_LABEL: ['red', 'dim'],
    MULTISIG_ADDRESS: ['yellow', 'dim'],
    CONTRACT_LABEL: ['blue', 'bright'],
    CONTRACT_ADDRESS: ['blue', 'dim'],
    OPERATOR_LABEL: ['green', 'bright'],
    OPERATOR_ADDRESS: ['green', 'dim'],
    UNKNOWN_ADDRESS: ['yellow', 'bright'],
  } as Styles

  const formatMultisig = (address: AccAddress, label: string) =>
    `[${logger.style(label, ...styles.MULTISIG_LABEL)}🧳${logger.style(address, ...styles.MULTISIG_ADDRESS)}]`

  const formatContract = (address: AccAddress, label: string) =>
    `[👝${logger.style(label, ...styles.CONTRACT_LABEL)}📜$${logger.style(address, ...styles.CONTRACT_ADDRESS)}]`

  // Shows up in terminal as single emoji (astronaut), but two emojis (adult + rocket) in some editors.
  // TODO: check portability, possibly just use adult emoji?
  //  https://emojiterra.com/astronaut-medium-skin-tone/  🧑🏽‍🚀
  //  https://emojipedia.org/pilot-medium-skin-tone  🧑🏽‍✈️

  const astronaut = '\uD83E\uDDD1\uD83C\uDFFD\u200D\uD83D\uDE80'
  //const pilot = '\uD83E\uDDD1\uD83C\uDFFD\u200D\u2708\uFE0F'

  const formatOperator = (address: AccAddress) =>
    `[${logger.style('operator', ...styles.CONTRACT_LABEL)}${astronaut}${logger.style(
      address,
      ...styles.CONTRACT_ADDRESS,
    )}]`

  const formatUnknown = (address: AccAddress) => `[👝${logger.style(address, ...styles.UNKNOWN_ADDRESS)}]`

  const instances = addressBook.instances

  if (address in instances) {
    if (instances[address].contractId == CONTRACT_LIST.MULTISIG) {
      return formatMultisig(address, instances[address].name)
    } else {
      return formatContract(address, instances[address].name)
    }
  } else if (address == addressBook.operator) {
    return formatOperator(address)
  } else {
    return formatUnknown(address)
  }
}

utils.fmtAddress = fmtAddress
