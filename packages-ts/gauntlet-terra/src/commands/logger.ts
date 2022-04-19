import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { AddressBook } from './addressBook'
import { assertions } from '@chainlink/gauntlet-core/dist/utils'

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
        return logger.formatMultisig(address, name)
      } else {
        return logger.formatContract(address, name)
      }
    } else if (address == this.addressBook.operator) {
      return logger.formatOperator(address)
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
