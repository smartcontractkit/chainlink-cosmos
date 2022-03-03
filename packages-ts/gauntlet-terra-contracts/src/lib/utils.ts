import { logger } from '@chainlink/gauntlet-core/dist/utils'
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

export function printDiff(existing, incoming) {
  for (const prop in existing) {
    const existingValue = existing[prop]
    const incomingValue = incoming[prop]

    if (Array.isArray(existingValue)) {
      logger.log(`${prop}: [`)

      for (const item of existingValue) {
        if (incomingValue.includes(item)) {
          logger.log(logger.style(`  ${item}`, 'yellow'))
        } else {
          logger.log(logger.style(`  ${item}`, 'red'))
        }
      }

      for (const item of incomingValue) {
        if (!existingValue.includes(item)) {
          logger.log(logger.style(`  ${item}`, 'green'))
        }
      }
      logger.log(`]`)
      continue
    }

    if (existingValue == incomingValue) {
      logger.log(`${prop}:`, logger.style(incomingValue, 'yellow'))
    } else {
      // todo: add \x1b[9m strikethrough styling option to the logger
      logger.log(`${prop}:`, logger.style(`\x1b[9m${existingValue}`, 'red'), logger.style(incomingValue, 'green'))
    }
  }
}

