import { wrapCommand as multisigWrapCommand } from './commands/multisig'
import Inspect from './commands/inspect'
import { multisig } from './commands/multisig'

const commands = [Inspect]

export { multisigWrapCommand, commands, multisig }
