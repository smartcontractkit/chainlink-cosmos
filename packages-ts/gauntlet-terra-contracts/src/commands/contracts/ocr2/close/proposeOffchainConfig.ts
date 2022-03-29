import { RDD } from '@chainlink/gauntlet-terra'
import { AbstractInstruction, instructionToCommand, BeforeExecute } from '../../../abstract/executionWrapper'
import { CATEGORIES } from '../../../../lib/constants'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { ContractInput } from '../proposeOffchainConfig'

export const EMPTY_CONFIG = Buffer.from([1, 2]).toString('base64')

type CommandInput = {
  proposalId: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {
    proposalId: flags.configProposal,
  }
}

const validateInput = (input: CommandInput): boolean => {
  if (!input.proposalId) throw new Error('Config Proposal not found')
  return true
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context, inputContext) => async () => {
  const rddContract = RDD.getContractFromRDD(RDD.getRDD(context.flags.rdd), context.contract)
  logger.info(`IMPORTANT: You are proposing an EMPTY configuration on the following contract:
    - Contract: ${rddContract.address} ${rddContract.description ? '- ' + rddContract.description : ''}
  `)
  await prompt('Continue?')
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    id: input.proposalId,
    offchain_config_version: 2,
    offchain_config: EMPTY_CONFIG,
  }
}

const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  examples: [
    'yarn gauntlet ocr2:propose_offchain_config:close --network=<NETWORK> --configProposal=<PROPOSAL_ID> <CONTRACT_ADDRESS>',
  ],
  instruction: {
    category: CATEGORIES.OCR,
    contract: 'ocr2',
    function: 'propose_offchain_config',
    subInstruction: 'close',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  beforeExecute,
}

export default instructionToCommand(instruction)
