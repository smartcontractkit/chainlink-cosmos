import { TerraCommand } from '@chainlink/gauntlet-terra'
import { MultisigTerraCommand } from '../contracts/multisig'

type COMMANDS = {
  custom: any[]
}

export default (commands: COMMANDS, slug: string) => {
  const slugs: string[] = slug.split(':')
  if (slugs.length < 3) {
    return null
  }
  const op: string = slugs.pop()!
  const command: any = commands.custom[slugs.join()]
  if (!!command) return undefined

  switch (op) {
    case 'multisig':
    case 'propose':
    case 'vote':
    case 'execute':
    case 'approve': // vote yes, then execute if threshold is reached
      return class Command extends MultisigTerraCommand {
        static id = slugs.join()
        multisigOp = () => {
          return op
        }
        commandType = () => {
          return command
        }
      }
    default:
      return undefined
  }
}
