import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { MultisigTerraCommand } from '../contracts/multisig'
import { Result, Command } from '@chainlink/gauntlet-core'

class EmptyCommand extends Command {
    constructor() {
        super({help : false}, [])
    }
}

export default (commands: any[], slug: string) : Command => {
  const slugs: string[] = slug.split(':')
  if (slugs.length < 3) {
    throw Error(`Command ${slugs.join(':')} not found`)
  }
  const op: string = slugs.pop()!
  const command: any = commands[slugs.join()]
  if (!!command) throw Error(`Command ${slugs.join(':')} not found`)

  switch (op) {
    case 'multisig':
    case 'propose':
    case 'vote':
    case 'execute':
    case 'approve': // vote yes, then execute if threshold is reached
      class WrappedCommand extends MultisigTerraCommand {
        static id = slugs.join()

        constructor(flags, args) {
            super(flags, args)
        }

        multisigOp = () => {
          return op
        }
        commandType = () => {
          return command
        }
      }

      let cmd = new EmptyCommand()
      let wc = Object.setPrototypeOf(WrappedCommand, EmptyCommand.prototype)
      wc = Object.assign(wc, cmd)
      return wc
    default:
        throw Error(`Command ${slugs.join(':')} not found`)
  }
}
