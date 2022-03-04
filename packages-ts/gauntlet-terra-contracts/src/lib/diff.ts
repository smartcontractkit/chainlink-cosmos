import { logger } from '@chainlink/gauntlet-core/dist/utils'

enum DIFF_PROPERTY_COLOR {
  ADDED = 'green',
  REMOVED = 'red',
  NO_CHANGE = 'yellow',
}

type DIFF_OPTIONS = {
  initialIndent?: string
  propertyName?: string
}

export function printDiff(existing: Object, incoming: Object, options?: DIFF_OPTIONS) {
  const { initialIndent = '', propertyName = 'Object' } = options || {}
  logger.log(initialIndent, propertyName, '{')
  const indent = initialIndent + '  '

  for (const prop of Object.keys(incoming)) {
    const existingProperty = existing?.[prop]
    const incomingProperty = incoming[prop]

    if (Array.isArray(incomingProperty)) {
      logger.log(indent, prop, ': [')
      const itemsIndent = indent + '  '

      for (const item of incomingProperty) {
        const itemStr = Buffer.isBuffer(item) ? item.toString('hex') : item
        if (existingProperty?.includes(item)) {
          logger.log(itemsIndent, logger.style(itemStr, DIFF_PROPERTY_COLOR.NO_CHANGE))
        } else {
          logger.log(itemsIndent, logger.style(itemStr, DIFF_PROPERTY_COLOR.ADDED))
        }
      }

      for (const item of existingProperty || []) {
        const itemStr = Buffer.isBuffer(item) ? item.toString('hex') : item
        if (!incomingProperty.includes(item)) {
          logger.log(itemsIndent, logger.style(itemStr, DIFF_PROPERTY_COLOR.REMOVED))
        }
      }
      logger.log(indent, `]`)
      continue
    }

    if (Buffer.isBuffer(incomingProperty)) {
      if (Buffer.compare(incomingProperty, existingProperty || Buffer.from('')) === 0) {
        logger.log(indent, `${prop}:`, logger.style(incomingProperty.toString('hex'), DIFF_PROPERTY_COLOR.NO_CHANGE))
      } else {
        logger.log(indent, `${prop}:`, logger.style(existingProperty?.toString('hex'), DIFF_PROPERTY_COLOR.REMOVED))
        logger.log(indent, `${prop}:`, logger.style(incomingProperty.toString('hex'), DIFF_PROPERTY_COLOR.ADDED))
      }
      continue
    }

    if (typeof incomingProperty === 'object') {
      printDiff(existingProperty, incomingProperty, {
        initialIndent: indent,
        propertyName: `${prop}:`,
      })
      continue
    }

    // plain property
    if (existingProperty == incomingProperty) {
      logger.log(indent, `${prop}:`, logger.style(incomingProperty, DIFF_PROPERTY_COLOR.NO_CHANGE))
    } else {
      logger.log(indent, `${prop}:`, logger.style(existingProperty, DIFF_PROPERTY_COLOR.REMOVED))
      logger.log(indent, `${prop}:`, logger.style(incomingProperty, DIFF_PROPERTY_COLOR.ADDED))
    }
  }

  logger.log(initialIndent, '}')
}
