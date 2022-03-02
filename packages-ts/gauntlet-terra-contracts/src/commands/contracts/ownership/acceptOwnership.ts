import { AbstractInstruction, BeforeExecute } from '../../abstract/executionWrapper'
import { CATEGORIES } from '../../../lib/constants'
import { CONTRACT_LIST } from '../../../lib/contracts'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { getContractFromRDD, getRDD } from '../../../lib/rdd'

type CommandInput = {}

type ContractInput = {}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => ({})
const makeContractInput = async (input: CommandInput): Promise<ContractInput> => ({})
const validateInput = (input: CommandInput): boolean => true

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context) => async (signer) => {
  const currentOwner = await context.query(context.contract, 'owner')
  if (!context.flags.rdd) {
    logger.warn('No RDD flag provided. Accepting ownership without RDD check')
    logger.info(`Accepting Ownership Transfer of contract with current owner ${currentOwner} to new owner `)
    await prompt('Continue?')
    return
  }
  const contract = getContractFromRDD(getRDD(context.flags.rdd), context.contract)
  logger.info(`Accepting Ownership Transfer of contract of type "${contract.type}":
    - Contract: ${contract.address} ${contract.description ? '- ' + contract.description : ''}
    - Current Owner: ${currentOwner}
    - Next Owner (Current signer): ${signer}
  `)
  await prompt('Continue?')
}

export const makeAcceptOwnershipInstruction = (contractId: CONTRACT_LIST) => {
  const acceptOwnershipInstruction: AbstractInstruction<CommandInput, ContractInput> = {
    instruction: {
      category: CATEGORIES.OWNERSHIP,
      contract: contractId,
      function: 'accept_ownership',
    },
    makeInput: makeCommandInput,
    validateInput: validateInput,
    makeContractInput: makeContractInput,
    beforeExecute,
  }

  return acceptOwnershipInstruction
}
