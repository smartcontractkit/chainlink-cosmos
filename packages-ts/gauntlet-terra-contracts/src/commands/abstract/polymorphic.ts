import { TerraCommand, TransactionResponse } from '@chainlink/gauntlet-terra'
import { MultisigTerraCommand } from '../contracts/multisig'
import { Result, Command } from '@chainlink/gauntlet-core'
import { logger } from '@chainlink/gauntlet-core/dist/utils'
import DeployLink from '../../commands/contracts/link/deploy'
import AbstractCommand from '.'
import { makeAbstractCommand } from '.'

class EmptyCommand extends Command {
  constructor() {
    super({ help: false }, [])
  }
}

type ICommandConstructor = (flags: any, args: string[]) => void

export default (commands: any, slug: string): Command => {
  const slugs: string[] = slug.split(':')
  if (slugs.length < 3) {
    throw Error(`Command ${slug} not found`)
  }
  const op: string = slugs.pop()!
  const instruction = slugs.join(':')

  const commandType = commands[instruction] ? commands[instruction] : AbstractCommand

  switch (op) {
    case 'multisig':
    case 'propose':
    case 'vote':
    case 'execute':
    case 'approve': // vote yes, then execute if threshold is reached
      class WrappedCommand extends MultisigTerraCommand {
        static id = instruction
        static commandType = commandType

        constructor(flags, args) {
          super(flags, args)
          if (commandType === AbstractCommand) {
            throw Error(`Command ${instruction} not found`)
            // TODO: get this working for abstract commands.  Something like:
            // this.command = await makeAbstractCommand(instruction, flags, args)
          } else {
            this.command = new commandType(flags, args)
          }
        }

        multisigOp = () => {
          return op
        }
      }

      // This is a temporary workaround for a bug in the type specification for findPolymorphic in @gauntlet-core.
      // At runtime, it's used as a constructor.  It must be callable and able to construct an actual command object
      // when passed flags & args, (ie, it should be a class).  But it's declared as if it were an instance of a type
      // satisfying ICommand, which will cause typescript to reject it during compilation if it's a class satisfying
      // the ICommand interface.  The only way to satisfy both constraints is for it to look like an instance at compile
      // time, and a class at runtime.    The workaround is to return something that is both at once:  add all properties
      // of an instance of EmtpyCommand to the class WrappedCommand and return the resulting hybrid.
      return Object.assign(WrappedCommand, new EmptyCommand())
    default:
      throw Error(`Command ${slugs.join(':')} not found`)
  }
}
