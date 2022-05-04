import { TerraCommand, logger } from '@chainlink/gauntlet-terra'
import { Middleware, Next, AddressBook } from '@chainlink/gauntlet-core'
import { CONTRACT_LIST } from './contracts'
import { AccAddress } from '@terra-money/terra.js'

const addressBooks = new Map<string, AddressBook>()

// Loads known addresses for deployed contracts from environment & wallet
//
// Commands on the same network share the same addressBook
// The logger also needs a reference to addressBook for logger.styleAddress(),
// but currently supports only one network
//
export const withAddressBook: Middleware = (c: TerraCommand, next: Next) => {
  const chainId = c.provider.config.chainID

  if (!addressBooks.has(chainId)) {
    addressBooks[chainId] = new AddressBook()
    addressBooks[chainId].setOperator(c.wallet.key.accAddress)

    const tryAddInstance = (id: CONTRACT_LIST, address: string | undefined, name?: string) => {
      if (!address) {
        logger.warn(`${id} not set in environment`)
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

    // TODO: extend logger for multi-chain
    logger.withAddressBook(addressBooks[chainId])
  }

  c.addressBook = addressBooks[chainId]

  return next()
}
