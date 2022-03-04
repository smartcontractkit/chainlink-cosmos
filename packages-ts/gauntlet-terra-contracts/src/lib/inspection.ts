import { Proto } from '@chainlink/gauntlet-core/dist/crypto'
import { BN } from '@chainlink/gauntlet-core/dist/utils'

export const toComparableLongNumber = (v: Long) => new BN(Proto.Protobuf.longToString(v)).toString()

export const toComparableNumber = (v: string | number | Long) => {
  // Proto encoding will ignore falsy values.
  if (!v) return '0'
  if (typeof v === 'string' || typeof v === 'number') return new BN(v).toString()
  return toComparableLongNumber(v)
}
