import { logger } from '@chainlink/gauntlet-core/dist/utils'
import { assertions } from '@chainlink/gauntlet-core/dist/utils'
import { assert } from 'console'
import { ContractId } from './types'

type Instance = {
  name: string
  contractId: ContractId
}

class AddressBook {
  operator: string
  instances: Map<string, Instance> // address => instance name

  constructor() {
    this.instances = new Map<string, Instance>()
  }

  setOperator(address: string) {
    this.operator = address
  }
  addInstance(contractId: ContractId, address: string, name?: string) {
    this.instances.set(address, { name, contractId } as Instance)
    logger.debug(`Using deployed instance of ${contractId}: ${name}=${address}`)
    return this
  }

  // format():  Automatically format terra addresses depending on contract type.
  //
  // Note:
  //  use(withAddressBook) middleware must be enabled for any commands calling this
  //
  // Example use:
  //  import addressBook as ab from '@chainlink/gauntlet-terra'
  //
  //  Use ${ab.format(address)} instead of ${address} in strings sent to console or log.
  //   - If it matches the address added with name='multisig', the address will show up
  //     as yellow and labelled "multisig".
  //   - If it matches a known contract address read from the environemnt (LINK, BILLING_ACCESS_CONTROLLER,... ),
  //     the address will be blue and labelled with the contract name( or "name" if specified )
  //   - Unknown addresses will remain unformmated
  format(address: string): string {
    assertions.assert(!!this.operator, `fmtAddress called on Command without "use withAddressBook"`)

    type COLOR = 'red' | 'blue' | 'yellow' | 'green'
    type INTENSITY = 'dim' | 'bright'
    type Style = COLOR | INTENSITY
    type Styles = {
      [key: string]: Style[]
    }
    const styles = {
      MULTISIG_LABEL: ['yellow', 'bright'],
      MULTISIG_ADDRESS: ['yellow', 'dim'],
      CONTRACT_LABEL: ['blue', 'bright'],
      CONTRACT_ADDRESS: ['blue', 'dim'],
      OPERATOR_LABEL: ['green', 'bright'],
      OPERATOR_ADDRESS: ['green', 'dim'],
    } as Styles

    const formatMultisig = (address: string, label: string): string =>
      `[${logger.style(label, ...styles.MULTISIG_LABEL)}🧳${logger.style(address, ...styles.MULTISIG_ADDRESS)}]`

    const formatContract = (address: string, label: string): string =>
      `[👝${logger.style(label, ...styles.CONTRACT_LABEL)}📜$${logger.style(address, ...styles.CONTRACT_ADDRESS)}]`

    // Shows up in terminal as single emoji (astronaut), but two emojis (adult + rocket) in some editors.
    // TODO: check portability, possibly just use adult emoji?
    //  https://emojiterra.com/astronaut-medium-skin-tone/  🧑🏽‍🚀
    //  https://emojipedia.org/pilot-medium-skin-tone  🧑🏽‍✈️

    const astronaut = '\uD83E\uDDD1\uD83C\uDFFD\u200D\uD83D\uDE80'
    //const pilot = '\uD83E\uDDD1\uD83C\uDFFD\u200D\u2708\uFE0F'

    const formatOperator = (address: string): string =>
      `[${logger.style('operator', ...styles.OPERATOR_LABEL)}${astronaut}${logger.style(
        address,
        ...styles.OPERATOR_ADDRESS,
      )}]`

    if (this.instances.has(address)) {
      const name = this.instances.get(address).name
      if (name == 'multisig') {
        return formatMultisig(address, name)
      } else {
        return formatContract(address, name)
      }
    } else if (address == this.operator) {
      return formatOperator(address)
    } else {
      return address
    }
  }
}

export default new AddressBook()
