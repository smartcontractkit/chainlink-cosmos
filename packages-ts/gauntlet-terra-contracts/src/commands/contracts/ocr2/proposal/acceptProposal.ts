import { Result } from '@chainlink/gauntlet-core'
import { TransactionResponse } from '@chainlink/gauntlet-terra'
import { AbstractInstruction, instructionToCommand } from '../../../abstract/executionWrapper'

type CommandInput = {
  proposalId: string
  digest: string
}

type ContractInput = {
  id: string
  digest: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {
    proposalId: flags.proposalId,
    digest: flags.digest,
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    id: input.proposalId,
    digest: Buffer.from(input.digest, 'hex').toString('base64'),
  }
}

const validateInput = (input: CommandInput): boolean => {
  if (!input.proposalId) throw new Error('A proposal ID is required. Provide it with --proposalId flag')
  return true
}

const afterExecute = async (response: Result<TransactionResponse>) => {
  console.log(response.data)
  return
}

// yarn gauntlet ocr2:accept_proposal --network=bombay-testnet --id=4 --digest=71e6969c14c3e0cd47d75da229dbd2f76fd0f3c17e05635f78ac755a99897a2f terra14nrtuhrrhl2ldad7gln5uafgl8s2m25du98hlx
const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    contract: 'ocr2',
    function: 'accept_proposal',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  afterExecute,
}

export default instructionToCommand(instruction)
