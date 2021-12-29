import { ICommand } from '@chainlink/gauntlet-core'
import { makeAbstractCommand } from '.'

type RawCommandInstruction = {
  instruction: string
  flags: any
  contract?: string
}

type Validation<CommandInput> = (input: CommandInput) => boolean
type CommandInputMaker<CommandInput> = (flags: any) => Promise<CommandInput>
type ContractInputMaker<CommandInput, ContractInput> = (input: CommandInput) => Promise<ContractInput>

const validateAddress = (string): boolean => {
  return true
}

/**
 *
 * @param rawCommand User interface with the basic info of the command to execute
 * @param makeCommandInput Should return the information the command needs to work
 * @param makeContractInput Transforms the Command Input into a valid input for the contract function
 * @param validateInput Validates the input the user provided
 * @returns An abstract command with clear instructions and a validated input
 */
export const abstractWrapper = async <CommandInput, ContractInput>(
  rawCommand: RawCommandInstruction,
  makeCommandInput: CommandInputMaker<CommandInput>,
  makeContractInput: ContractInputMaker<CommandInput, ContractInput>,
  validateInput: Validation<CommandInput>,
): Promise<ICommand | undefined> => {
  const commandInput = await makeCommandInput(rawCommand.flags)
  try {
    validateInput(commandInput)
    validateAddress(rawCommand.contract)
    const input = await makeContractInput(commandInput)
    return await makeAbstractCommand(rawCommand.instruction, rawCommand.flags, [rawCommand.contract || ''], input)
  } catch (e) {
    throw new Error(`Error validating Input: ${e.message}`)
  }
}
