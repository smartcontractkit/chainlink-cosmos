import { TerraCommand, AddressBook, logger } from '@chainlink/gauntlet-terra'
import { Middleware, Next } from '@chainlink/gauntlet-core'
import { CONTRACT_LIST } from './contracts'
import { AccAddress } from '@terra-money/terra.js'

const addressBooks = new Map<string, AddressBook>()

// Loads known addresses for deployed contracts from environment
// and local operator's wallet from previous middleware in same
// TerraCommand.
//
// There is one addressBook for each network. Each addressBook is
// populated only once, but each TerraCommand on the same network is
// given a reference to it.  The logger is also given a copy of the
// addressBook, and used by logger.styleAddress() to label and stylize
// addresses by contract type.
//
export const withAddressBook: Middleware = (c: TerraCommand, next: Next) => {
  const chainId = c.provider.config.chainID

  if (!addressBooks.has(chainId)) {
    addressBooks[chainId] = new AddressBook()
    addressBooks[chainId].setOperator(c.wallet.key.accAddress)

    const tryAddInstance = (id: CONTRACT_LIST, address: string | undefined, name?: string) => {
      if (!address) {
        console.warn(`${address} not set in environment`)
      } else if (!AccAddress.validate(address)) {
        throw new Error(`Read invalid contract address ${address} for ${id} contract from env`)
      } else {
        addressBooks[chainId].addInstance(id, address, name)
      }
    }

    // Addresses of deployed instances read from env vars
    tryAddInstance(CONTRACT_LIST.CW20_BASE, process.env['LINK'], 'link')
    tryAddInstance(CONTRACT_LIST.ACCESS_CONTROLLER, process.env['BILLING_ACCESS_CONTROLLER'], 'billing_access')
    tryAddInstance(CONTRACT_LIST.ACCESS_CONTROLLER, process.env['REQUESTER_ACCESS_CONTROLLER'], 'requester_access')
    tryAddInstance(CONTRACT_LIST.CW4_GROUP, process.env['CW4_GROUP'])
    tryAddInstance(CONTRACT_LIST.MULTISIG, process.env['CW3_FLEX_MULTISIG'], 'multisig')

    // TODO: currently the logger itself only supports one network at a time.
    // To fully support multichain, we could have either:
    //   1. the logger store a collection of address books by network, where
    //      logger.withAddressBook(addressBook) becomes logger.addAddressBook(addressBook) and
    //      logger.styleAddress(address) becomes logger.styleAddress(chainId, address).
    //  or
    //   2. multiple logger instances, also segmented by network, in which case it would
    //      need to be called as this.logger(address) from each command.
    //
    //  Both of these seem a bit awkward, but I think the only other option would be to move
    //  styleAddress from the TerraLogger back into AddressBook.  Leaving this decision
    //  for the future when/if the rest of gauntlet is adapted for multichain

    logger.withAddressBook(addressBooks[chainId])
  }

  c.addressBook = addressBooks[chainId]

  return next()
}
