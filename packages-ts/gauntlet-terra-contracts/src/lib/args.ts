// Doesn't work as expected - https://github.com/smartcontractkit/chainlink-terra/issues/199
// As a workaround, default RDD flag is set from env in gauntlet-terra/src/lib/rdd.ts
export const defaultFlags = {
  delta: 'delta.json',
  codeIdsPath: './codeIds',
  artifactsPath: './artifacts',
}
