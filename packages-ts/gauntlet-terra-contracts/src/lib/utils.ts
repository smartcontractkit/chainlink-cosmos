import { bech32 } from 'bech32'

// https://docs.terra.money/docs/develop/sdks/terra-js/common-examples.html
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

export function deepCopy<T>(source: T): T {
  return Buffer.isBuffer(source)
    ? Buffer.from(source)
    : Array.isArray(source)
    ? source.map((item) => this.deepCopy(item))
    : source instanceof Date
    ? new Date(source.getTime())
    : source && typeof source === 'object'
    ? Object.getOwnPropertyNames(source).reduce((o, prop) => {
        Object.defineProperty(o, prop, Object.getOwnPropertyDescriptor(source, prop)!)
        o[prop] = this.deepCopy((source as { [key: string]: any })[prop])
        return o
      }, Object.create(Object.getPrototypeOf(source)))
    : (source as T)
}
