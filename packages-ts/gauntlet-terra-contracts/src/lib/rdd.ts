import { existsSync, readFileSync } from 'fs'
import { join } from 'path'

export const getRDD = (path: string, fileDescription: string = 'RDD'): any => {
  let pathToUse
  // test whether the file exists as a relative path or an absolute path
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
