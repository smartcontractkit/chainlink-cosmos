import { bech32 } from 'bech32';

type GroupMember = {
  addr: string,
  weight: number
}

type Duration = {
  height?: number,   // block height
  time?: number     // length of time in seconds
}

function validDuration(d: Duration): boolean {
  if ((typeof d.height) === 'number' && (typeof d.time) === 'undefined') {
    return d.height! >= 0
  } else if ((typeof d.height) === 'undefined' && (typeof d.time) === 'number') {
    return d.time! >= 0
  } else {
    return false // must have either height or time, but not both
  }
}

type AbsCount = {
  weight: number
}

type AbsPerc = {
  percentage: number
}

type ThreshQuorum = {
  threshold : number,
  quorum : number
}

type Threshold = {
  absolute_count? : AbsCount,
  absolute_percentage? : AbsPerc,
  threshold_quorum? : ThreshQuorum
}

function validThreshold(t: Threshold): boolean {
  // TODO
  return true
}

// validate a terra address
//  from: https://docs.terra.money/docs/develop/sdks/terra-js/common-examples.html
function validAddr(address) {
  try {
    const { prefix: decodedPrefix } = bech32.decode(address); // throw error if checksum is invalid
    // verify address prefix
    return decodedPrefix === 'terra';
  } catch {
    // invalid checksum
    return false;
  }
}

export { GroupMember, Duration, validDuration, Threshold, validThreshold, validAddr }
