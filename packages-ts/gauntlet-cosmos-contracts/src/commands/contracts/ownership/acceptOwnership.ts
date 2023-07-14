import { AbstractInstruction, BeforeExecute } from '../../abstract/executionWrapper'
import { RDD, logger, AccAddress } from '@chainlink/gauntlet-cosmos'
import { CATEGORIES } from '../../../lib/constants'
import { CONTRACT_LIST } from '../../../lib/contracts'

type CommandInput = {}

type ContractInput = {}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => ({})
const makeContractInput = async (input: CommandInput): Promise<ContractInput> => ({})
const validateInput = (input: CommandInput): boolean => true

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context) => async (signer) => {
  const currentOwner: AccAddress = await context.provider.queryContractSmart(context.contract, 'owner' as any)
  const contract = RDD.getContractFromRDD(RDD.getRDD(context.flags.rdd), context.contract)
  logger.info(`Accepting Ownership Transfer of contract of type "${contract.type}":
    - Contract: ${logger.styleAddress(contract.address)} ${contract.description ? '- ' + contract.description : ''}
    - Current Owner: ${logger.styleAddress(currentOwner)}
    - Next Owner (Current signer): ${logger.styleAddress(signer)}
  `)
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
