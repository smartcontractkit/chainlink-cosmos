import { RDD } from '@chainlink/gauntlet-cosmos'
import { DeployOCR2, DeployOCR2Input } from '@chainlink/gauntlet-contracts-ocr2'
import { instructionToCommand, AbstractInstruction } from '../../abstract/executionWrapper'
import { CATEGORIES } from '../../../lib/constants'

export interface CommandInput extends DeployOCR2Input {
  requesterAccessController: string
}

type ContractInput = {
  billing_access_controller: string
  requester_access_controller: string
  link_token: string
  decimals: number
  description: string
  max_answer: string
  min_answer: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  if (flags.rdd) {
    const rdd = RDD.getRDD(flags.rdd)
    const contract = args[0]
    const aggregator = rdd.contracts[contract]
    return {
      maxAnswer: aggregator.maxSubmissionValue,
      minAnswer: aggregator.minSubmissionValue,
      decimals: aggregator.decimals,
      description: aggregator.name,
      billingAccessController: flags.billingAccessController || process.env.BILLING_ACCESS_CONTROLLER || '',
      requesterAccessController: flags.requesterAccessController || process.env.REQUESTER_ACCESS_CONTROLLER || '',
      linkToken: flags.link || process.env.LINK || '',
    }
  }
  flags.minSubmissionValue = parseInt(flags.minSubmissionValue)
  flags.maxSubmissionValue = parseInt(flags.maxSubmissionValue)
  flags.decimals = parseInt(flags.decimals)

  return {
    ...DeployOCR2.makeUserInput(flags as DeployOCR2Input, args, process.env),
    billingAccessController: flags.billingAccessController || process.env.BILLING_ACCESS_CONTROLLER || '',
    requesterAccessController: flags.requesterAccessController || process.env.REQUESTER_ACCESS_CONTROLLER || '',
    linkToken: flags.link || process.env.LINK || '',
  } as CommandInput
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    billing_access_controller: input.billingAccessController,
    requester_access_controller: input.requesterAccessController,
    link_token: input.linkToken,
    decimals: input.decimals,
    description: input.description,
    max_answer: input.maxAnswer.toString(),
    min_answer: input.minAnswer.toString(),
  }
}

const validateInput = (input: CommandInput): boolean => {
  return true
}

const deployInstruction: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    category: CATEGORIES.OCR,
    contract: 'ocr2',
    function: 'deploy',
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
}

export default instructionToCommand(deployInstruction)
