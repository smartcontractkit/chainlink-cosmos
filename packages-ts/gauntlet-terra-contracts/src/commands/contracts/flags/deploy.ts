import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { bech32 } from 'bech32'
import { AbstractInstruction, instructionToCommand } from '../../abstract/wrapper'

type CommandInput = {
    raisingAccessController: string,
    loweringAccessController: string
}

type ContractInput = {
    raising_access_controller: string,
    lowering_access_controller: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  return {
    raisingAccessController: flags.raisingAccessController,
    loweringAccessController: flags.loweringAccessController,
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    raising_access_controller: input.raisingAccessController,
    lowering_access_controller: input.loweringAccessController,
  }
}

const validateInput = (input: CommandInput): boolean => {
  validateAddress(input.raisingAccessController)
  validateAddress(input.loweringAccessController)

  return true
}

const validateAddress = (address: string): boolean => {
  const { prefix: decodedPrefix } = bech32.decode(address) // throws error if checksum is invalid which will fail validation

  // verify address prefix
  if (decodedPrefix !== 'terra') {
    throw new Error(`Invalid address prefix (expecteed: 'terra')`)
  }

  return true
}

const deploy: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    contract: 'flags',
    function: 'deploy',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(deploy)
