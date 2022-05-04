// TODO: This function should be transfered to gauntlet-core repo.
export function dateFromUnix(unixTimestamp: number): Date {
  return new Date(unixTimestamp * 1000)
}

export const hexToBase64 = (s: string): string => Buffer.from(s, 'hex').toString('base64')
