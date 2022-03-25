import { TerraCommand, AddressBook, logger } from '@chainlink/gauntlet-terra'
import { Middleware, Next } from '@chainlink/gauntlet-core'
import { CONTRACT_LIST } from './contracts'
import { AccAddress } from '@terra-money/terra.js'

// Loads known addresses for deployed contracts from environment
// and local operator's wallet from previous middleware in same
// TerraCommand. These are used by logger.styleAddress() to properly
// label addresses when displayed.
export const withAddressBook: Middleware = (c: TerraCommand, next: Next) => {
  c.addressBook = new AddressBook()
  c.addressBook.setOperator(c.wallet.key.accAddress)

  const tryAddInstance = (id: CONTRACT_LIST, address: string | undefined, name?: string) => {
    if (!address) {
      console.warn(`${address} not set in environment`)
    } else if (!AccAddress.validate(address)) {
      throw new Error(`Read invalid contract address ${address} for ${id} contract from env`)
    } else {
      c.addressBook.addInstance(id, address, name)
    }
  }

  // Addresses of deployed instances read from env vars
  tryAddInstance(CONTRACT_LIST.CW20_BASE, process.env['LINK'], 'link')
  tryAddInstance(CONTRACT_LIST.ACCESS_CONTROLLER, process.env['BILLING_ACCESS_CONTROLLER'], 'billing_access')
  tryAddInstance(CONTRACT_LIST.ACCESS_CONTROLLER, process.env['REQUESTER_ACCESS_CONTROLLER'], 'requester_access')
  tryAddInstance(CONTRACT_LIST.CW4_GROUP, process.env['CW4_GROUP'])
  tryAddInstance(CONTRACT_LIST.MULTISIG, process.env['CW3_FLEX_MULTISIG'], 'multisig')

  logger.withAddressBook(c.addressBook)

  return next()
}
