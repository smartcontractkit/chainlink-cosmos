import { startWasmdAndUpload } from '../utils'

/**
 * Start Wasmd and Upload Base Contracts.
 *
 * Intended for use during local debugging / testing
 *
 * Workflow:
 * 1. yarn test:debug:wasmd-up
 * 2. yarn test:debug
 */
// yarn test:debug:wasmd-up
const run = async () => {
  await startWasmdAndUpload()
}

run()
