import { TerraCommand } from '@chainlink/gauntlet-terra'
import { Middleware, Next } from '@chainlink/gauntlet-core'
import { addressBook, CONTRACT_LIST } from './contracts'

// Loads known addresses for deployed addressBook from environment
// and local operator's wallet from previous middleware in same
// TerraCommand. These are used by fmtAddress in utils.ts to
// properly label addresses when displayed.
export const withAddressBook: Middleware = (c: TerraCommand, next: Next) => {
  addressBook.addOperator(c.wallet.key.accAddress)

  // Addresses of deployed instances read from env vars
  addressBook.addInstance(CONTRACT_LIST.CW20_BASE, 'LINK', 'link')
  addressBook.addInstance(CONTRACT_LIST.ACCESS_CONTROLLER, 'BILLING_ACCESS_CONTROLLER', 'billing_access')
  addressBook.addInstance(CONTRACT_LIST.ACCESS_CONTROLLER, 'REQUESTER_ACCESS_CONTROLLER', 'requester_access')
  addressBook.addInstance(CONTRACT_LIST.CW4_GROUP, 'CW4_GROUP')
  addressBook.addInstance(CONTRACT_LIST.MULTISIG, 'CW3_FLEX_MULTISIG', 'multisig')

  return next()
}
