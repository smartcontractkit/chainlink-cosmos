import { existsSync, readFileSync } from 'fs'
import { join } from 'path'

export const getRDD = (path: string, fileDescription: string = 'RDD'): any => {
  path = path || process.env.RDD
  if (!path) {
    throw new Error(`No reference data directory specified!  Must pass in the '--rdd' flag or set the 'RDD' env var`)
  }

  // test whether the file exists as a relative path or an absolute path
  let pathToUse
  if (existsSync(path)) {
    pathToUse = path
  } else if (existsSync(join(process.cwd(), path))) {
    pathToUse = join(process.cwd(), path)
  } else {
    throw new Error(`Could not find the ${fileDescription}. Make sure you provided a valid ${fileDescription} path`)
  }

  try {
    const buffer = readFileSync(pathToUse, 'utf8')
    return JSON.parse(buffer.toString())
  } catch (e) {
    throw new Error(
      `An error ocurred while parsing the ${fileDescription}. Make sure you provided a valid ${fileDescription} path`,
    )
  }
}

export enum CONTRACT_TYPES {
  PROXY = 'proxies',
  FLAG = 'flags',
  ACCESS_CONTROLLER = 'accessControllers',
  CONTRACT = 'contracts',
  VALIDATOR = 'validators',
}

export type RDDContract = {
  type: CONTRACT_TYPES
  contract: any
  address: string
  description?: string
}

export const getContractFromRDD = (rdd: any, address: string): RDDContract => {
  return Object.values(CONTRACT_TYPES).reduce((agg, type) => {
    const content = rdd[type]?.[address]
    if (content) {
      return {
        type,
        contract: content,
        address,
        ...((type === CONTRACT_TYPES.CONTRACT || type === CONTRACT_TYPES.PROXY) && { description: content.name }),
      }
    }
    return agg
  }, {} as RDDContract)
}
