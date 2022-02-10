export type GroupMember = {
  addr: string
  weight: number
}

export type Duration = {
  height?: number // block height
  time?: number // length of time in seconds
}

export function isValidDuration(d: Duration): boolean {
  if (typeof d === 'undefined') {
    return false
  } else if (typeof d.time === 'undefined' && typeof d.height === 'number') {
    return d.height! >= 0
  } else if (typeof d.height === 'undefined' && typeof d.time === 'number') {
    return d.time! >= 0
  } else {
    return false // must have either height or time, but not both
  }
}

export type AbsCount = {
  weight: number
}

export type AbsPerc = {
  percentage: number
}

export type ThreshQuorum = {
  threshold: number
  quorum: number
}

export type Threshold = {
  absolute_count?: AbsCount
  absolute_percentage?: AbsPerc
  threshold_quorum?: ThreshQuorum
}

export function isValidThreshold(t: Threshold): boolean {
  // TODO
  return true
}
