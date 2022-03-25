import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { AddressBook } from './addressBook'
import { assertions } from '@chainlink/gauntlet-core/dist/utils'

type COLOR = 'red' | 'blue' | 'yellow' | 'green'
type INTENSITY = 'dim' | 'bright'
type Style = COLOR | INTENSITY
type Styles = { [id: string]: [color: COLOR, intensity: INTENSITY] }
const styles = {
  MULTISIG_LABEL: ['yellow', 'bright'],
  MULTISIG_ADDRESS: ['yellow', 'dim'],
  CONTRACT_LABEL: ['blue', 'bright'],
  CONTRACT_ADDRESS: ['blue', 'dim'],
  OPERATOR_LABEL: ['green', 'bright'],
  OPERATOR_ADDRESS: ['green', 'dim'],
} as Styles

// Shows up in terminal as single emoji (astronaut), but two emojis (adult + rocket) in some editors.
// TODO: check portability, possibly just use adult emoji?
//  https://emojiterra.com/astronaut-medium-skin-tone/  🧑🏽‍🚀
const astronaut = '\uD83E\uDDD1\uD83C\uDFFD\u200D\uD83D\uDE80'

const formatMultisig = (address: string, label: string): string =>
  `[${logger.style(label, ...styles.MULTISIG_LABEL)}🧳${logger.style(address, ...styles.MULTISIG_ADDRESS)}]`

const formatContract = (address: string, label: string): string =>
  `[👝${logger.style(label, ...styles.CONTRACT_LABEL)}📜$${logger.style(address, ...styles.CONTRACT_ADDRESS)}]`

const formatOperator = (address: string): string =>
  `[${logger.style('operator', ...styles.OPERATOR_LABEL)}${astronaut}${logger.style(
    address,
    ...styles.OPERATOR_ADDRESS,
  )}]`

export class TerraLogger {
  addressBook: AddressBook

  withAddressBook(addressBook: AddressBook) {
    this.addressBook = addressBook
  }

  // logger.styleAddress():  Format a terra addresses depending on contract type.
  // Usage:
  //
  // 1. Call logger.withAddressBook(addressBook) in middleware
  //
  // 2. import { logger } from '@gauntlet-terra/dist/commands/logger'
  //
  // 3. Use ${logger.styleAddress(address)} instead of ${address} in strings sent to console or log.
  //   - If it matches the address added with name='multisig', the address will show up
  //     as yellow and labelled "multisig".
  //   - If it matches a known contract address read from the environemnt (LINK, BILLING_ACCESS_CONTROLLER,... ),
  //     the address will be blue and labelled with the contract name( or "name" if specified )
  //   - Unknown addresses will remain unformmated
  styleAddress(address: string): string {
    if (!this.addressBook) {
      logger.warn(`TerraLogger: styleAddress called before calling withAddressBook!`)
      return address
    }

    if (this.addressBook.instances.has(address)) {
      const name = this.addressBook.instances.get(address).name
      if (name == 'multisig') {
        return formatMultisig(address, name)
      } else {
        return formatContract(address, name)
      }
    } else if (address == this.addressBook.operator) {
      return formatOperator(address)
    } else {
      return address
    }
  }
}

const terraLogger = new TerraLogger()

export default {
  table: logger.table,
  log: logger.log,
  info: logger.info,
  warn: logger.warn,
  success: logger.success,
  error: logger.error,
  loading: logger.loading,
  line: logger.line,
  debug: logger.debug,
  time: logger.time,
  style: logger.style,
  styleAddress: (address: string) => terraLogger.styleAddress(address),
  withAddressBook: (addressBook: AddressBook) => terraLogger.withAddressBook(addressBook),
}
