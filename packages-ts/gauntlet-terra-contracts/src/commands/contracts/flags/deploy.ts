import { BN } from '@chainlink/gauntlet-core/dist/utils'
import { bech32 } from 'bech32'
import { AccAddress } from '@terra-money/terra.js'
import { AbstractInstruction, instructionToCommand } from '../../abstract/wrapper'

type CommandInput = {
  raisingAccessController: string
  loweringAccessController: string
}

type ContractInput = {
  raising_access_controller: string
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
  if (!AccAddress.validate(input.raisingAccessController)) throw new Error(`Invalid raisingAccessController address`)
  if (!AccAddress.validate(input.loweringAccessController)) throw new Error(`Invalid loweringAccessController address`)

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
