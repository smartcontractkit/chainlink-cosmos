import Ajv from 'ajv'
import JTD from 'ajv/dist/jtd'
import { bech32 } from 'bech32'

const ajv = new Ajv().addFormat('uint8', (value: any) => !isNaN(value))

ajv.addFormat('uint64', {
  type: 'number',
  validate: (x) => !isNaN(x),
})

ajv.addFormat('uint32', {
  type: 'number',
  validate: (x) => !isNaN(x),
})

export default ajv

const jtd = new JTD()
export { jtd }

// validate a terra address
//  from: https://docs.terra.money/docs/develop/sdks/terra-js/common-examples.html
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
