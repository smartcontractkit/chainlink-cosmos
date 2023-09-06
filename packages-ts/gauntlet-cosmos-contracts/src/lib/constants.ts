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
export const DEFAULT_CWPLUS_VERSION = 'v0.16.0'

export const ORACLES_MAX_LENGTH = 31

export const CW20_BASE_CODE_IDs = {
  mainnet: 3,
  'testnet-bombay': 148,
  'testnet-bombay-internal': 148,
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

export const UCOSM_DECIMALS = 6

export const EMPTY_TRANSMITTERS = [
  'wasm1hft9sxhx7d7furw9y0rjxu4hfsm76ehkman78g',
  'wasm10f0wy3fs6ex395ylturr0hv03m3cjcjpy4ux6x',
  'wasm1jv45uny4kuyeecgzw5xftkr7nssdj5e56ajchs',
  'wasm1n947av78pavcs9tpp79me30gk6rfqhnam9gsls',
]
