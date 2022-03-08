import { AbstractInstruction, BeforeExecute } from '../../abstract/executionWrapper'
import { RDD } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../../lib/constants'
import { CONTRACT_LIST } from '../../../lib/contracts'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'

type CommandInput = {}

type ContractInput = {}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => ({})
const makeContractInput = async (input: CommandInput): Promise<ContractInput> => ({})
const validateInput = (input: CommandInput): boolean => true

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context) => async (signer) => {
  const currentOwner = await context.provider.wasm.contractQuery(context.contract, 'owner' as any)
  if (!context.flags.rdd) {
    throw new Error(`No RDD flag provided!`)
  }
  const contract = RDD.getContractFromRDD(RDD.getRDD(context.flags.rdd), context.contract)
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
