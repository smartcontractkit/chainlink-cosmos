import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { AddressBook } from '@chainlink/gauntlet-core/dist/commands/addressBook'
import { assertions } from '@chainlink/gauntlet-core/dist/utils'
import { Logger } from '@chainlink/gauntlet-core/dist/utils/logger'

// TODO: instatntiate in terra-gauntlet-contracts instead of here?
const terraLogger = new Logger()

export default {
  styleAddress: (address: string) => terraLogger.styleAddress(address),
  withAddressBook: (addressBook: AddressBook) => terraLogger.withAddressBook(addressBook),
  ...logger,
}
