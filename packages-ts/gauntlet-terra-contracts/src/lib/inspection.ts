import { Proto } from '@chainlink/gauntlet-core/dist/crypto'
import { BN } from '@chainlink/gauntlet-core/dist/utils'

export const toComparableNumber = (v: string | number) => new BN(v).toString()
export const toComparableLongNumber = (v: Long) => new BN(Proto.Protobuf.longToString(v)).toString()
export const wrappedComparableLongNumber = (v: any) => {
  // Proto encoding will ignore falsy values.
  if (!v) return '0'
  return toComparableLongNumber(v)
}
