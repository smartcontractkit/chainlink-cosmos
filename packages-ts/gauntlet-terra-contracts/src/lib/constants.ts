import { BN } from '@chainlink/gauntlet-core/dist/utils'

export const enum CATEGORIES {
  OWNERSHIP = 'Ownership',
  PROXIES = 'Proxies',
  TOOLING = 'Tooling',
  V3 = 'V3',
  LINK = 'LINK',
  FLAGS = 'Flags',
  OCR = 'OCR',
  ACCESS_CONTROLLER = 'Access Controller',
  MULTISIG = 'Multisig',
  DEVIATION_FLAGGING_VALIDATOR = 'Devaiation Flagging Validator',
  WALLET = 'Wallet',
}

export const DEFAULT_RELEASE_VERSION = 'local'
export const DEFAULT_CWPLUS_VERSION = 'v0.9.1'

export const ORACLES_MAX_LENGTH = 31

export const CW20_BASE_CODE_IDs = {
  mainnet: 3,
  local: 32,
  'testnet-bombay': 148,
}

export const CW4_GROUP_CODE_IDs = {
  mainnet: -1,
  local: -1,
  'testnet-bombay': 36895,
}

export const CW3_FLEX_MULTISIG_CODE_IDs = {
  mainnet: -1,
  local: -1,
  'testnet-bombay': 36059,
}

export const TOKEN_DECIMALS = 18
export const TOKEN_UNIT = new BN(10).pow(new BN(TOKEN_DECIMALS))

export const ULUNA_DECIMALS = 6

export const EMPTY_TRANSMITTERS = [
  'terra1deadlc2heq806uyw743z4d79dj7rcg7hga852t',
  'terra1deadmx4kksqqcagp5d02yc44390c5tv8nmfgav',
  'terra1deadnd9uqqwx7wyq4jsgrv7h5shl5sfqm007vd',
  'terra1deaddfmsng0s0gzahytn8m6tv858a4mzg0t0gq',
]
