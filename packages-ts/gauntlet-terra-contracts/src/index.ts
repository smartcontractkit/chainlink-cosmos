import { executeCLI } from '@chainlink/gauntlet-core'
import { existsSync } from 'fs'
import path from 'path'
import Terra from './commands'
import { makeAbstractCommand } from './commands/abstract'
import { defaultFlags } from './lib/args'

const commands = {
  custom: [...Terra],
  loadDefaultFlags: () => defaultFlags,
  abstract: {
    findPolymorphic: () => undefined,
    makeCommand: makeAbstractCommand,
  },
}

;(async () => {
  try {
    const networkPossiblePaths = ['./networks', './packages-ts/gauntlet-terra-contracts/networks']
    const networkPath = networkPossiblePaths.filter((networkPath) =>
      existsSync(path.join(process.cwd(), networkPath)),
    )[0]
    await executeCLI(commands, networkPath)
  } catch (e) {
    console.log('Terra Command execution error', e)
  }
})()
