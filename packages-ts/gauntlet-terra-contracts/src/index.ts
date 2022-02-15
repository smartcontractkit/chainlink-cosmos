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
    const networkPossiblePaths = ['./packages-ts/gauntlet-terra-contracts/networks']
    const networkPath = networkPossiblePaths.filter((networkPath) =>
      existsSync(path.join(process.cwd(), networkPath)),
    )[0]
    /* const result = */ await executeCLI(commands, networkPath)
    // TODO: save report just as on Solana
    // if (result) io.saveJSON(result, 'report')
  } catch (e) {
    console.log(e)
    console.log('Terra Command execution error', e.message)
    process.exitCode = 1
  }
})()
