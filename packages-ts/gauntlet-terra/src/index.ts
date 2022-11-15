import TerraCommand from './commands/internal/terra'
import { waitExecute } from './lib/execute'
import { TransactionResponse } from './commands/types'
import * as constants from './lib/constants'
import * as providerUtils from './lib/provider'
import * as RDD from './lib/rdd'
import { AddressBook } from '@chainlink/gauntlet-core'
import logger from './commands/logger'
import { fromBech32 } from '@cosmjs/encoding'

export { TerraCommand, waitExecute, TransactionResponse, constants, providerUtils, RDD, AddressBook, logger }

// TODO: just use normalizeBech32() instead of this type
function checkPrefixAndLength(
  prefix: string,
  data: string,
  length: number
): boolean {
  try {
    const vals = fromBech32(data);
    return vals.prefix === prefix && data.length == length;
  } catch (e) {
    return false;
  }
}export declare type AccAddress = string;
export namespace AccAddress {
  /**
   * Checks if a string is a valid Terra account address.
   *
   * @param data string to check
   */
  export function validate(data: string): boolean {
    // 44 for normal account and 64 for contract account
    return (
      checkPrefixAndLength('terra', data, 44) ||
      checkPrefixAndLength('terra', data, 64)
    );
  }
}

export { CosmWasmClient as Client } from '@cosmjs/cosmwasm-stargate'
