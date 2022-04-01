import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { AddressBook } from './addressBook'
import { assertions } from '@chainlink/gauntlet-core/dist/utils'

type COLOR = 'red' | 'green' | 'blue' | 'yellow' | 'cyan' | 'magenta'
type INTENSITY = 'dim' | 'bright'
type Style = COLOR | INTENSITY
type Styles = { [id: string]: [color: COLOR, intensity: INTENSITY] }
const styles = {
  MULTISIG_LABEL: ['cyan', 'bright'],
  MULTISIG_ADDRESS: ['cyan', 'dim'],
  CONTRACT_LABEL: ['blue', 'bright'],
  CONTRACT_ADDRESS: ['blue', 'dim'],
  OPERATOR_LABEL: ['green', 'bright'],
  OPERATOR_ADDRESS: ['green', 'dim'],
} as Styles

const astronaut = '\uD83E\uDDD1\uD83C\uDFFD\u200D\uD83D\uDE80'

const formatMultisig = (address: string, label: string): string =>
  `🧳 ${logger.style(label, ...styles.MULTISIG_LABEL)}: ${logger.style(address, ...styles.MULTISIG_ADDRESS)}`

const formatContract = (address: string, label: string): string =>
  `📜 ${logger.style(label, ...styles.CONTRACT_LABEL)}: ${logger.style(address, ...styles.CONTRACT_ADDRESS)}`

const formatOperator = (address: string): string =>
  `🧑🏽 ${logger.style('operator', ...styles.OPERATOR_LABEL)}: ${logger.style(address, ...styles.OPERATOR_ADDRESS)}`

export class TerraLogger {
  addressBook: AddressBook

  withAddressBook(addressBook: AddressBook) {
    this.addressBook = addressBook
  }

  // Example:  logger.info(`Destination address is ${logger.styleAddress(address)}`)
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

// TODO: instatntiate in terra-gauntlet-contracts instead of here?
const terraLogger = new TerraLogger()

export default {
  styleAddress: (address: string) => terraLogger.styleAddress(address),
  withAddressBook: (addressBook: AddressBook) => terraLogger.withAddressBook(addressBook),
  ...logger,
}
