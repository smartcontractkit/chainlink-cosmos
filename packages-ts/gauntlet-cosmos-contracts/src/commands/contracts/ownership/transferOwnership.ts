import { RDD, logger, AccAddress } from '@chainlink/gauntlet-cosmos'
import { AbstractInstruction, BeforeExecute } from '../../abstract/executionWrapper'
import { CATEGORIES } from '../../../lib/constants'
import { CONTRACT_LIST } from '../../../lib/contracts'
import { RDDContract } from '@chainlink/gauntlet-cosmos/dist/lib/rdd'

type CommandInput = {
  to: string
}

type ContractInput = {
  to: string
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  return {
    to: flags.to,
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    to: input.to,
  }
}

const validateInput = (input: CommandInput): boolean => {
  if (!AccAddress.validate(input.to)) {
    throw new Error(`Invalid proposed owner address!`)
  }

  return true
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context, input) => async () => {
  const currentOwner = await context.provider.queryContractSmart(context.contract, 'owner' as any)
  let contract: RDDContract | null = null
  if (context.flags.rdd) {
    contract = RDD.getContractFromRDD(RDD.getRDD(context.flags.rdd), context.contract)
  }
  logger.info(`Proposing Ownership Transfer of contract" ${contract ? ' of type' + contract.type : ''}":
    - Contract: ${context.contract} ${contract?.description ? '- ' + contract.description : ''}
    - Current Owner: ${currentOwner}
    - Next Owner: ${logger.styleAddress(input.contract.to)}
  `)
}

export const makeTransferOwnershipInstruction = (contractId: CONTRACT_LIST) => {
  const transferOwnershipInstruction: AbstractInstruction<CommandInput, ContractInput> = {
    instruction: {
      category: CATEGORIES.OWNERSHIP,
      contract: contractId,
      function: 'transfer_ownership',
    },
    makeInput: makeCommandInput,
    validateInput: validateInput,
    makeContractInput: makeContractInput,
    beforeExecute,
  }

  return transferOwnershipInstruction
}
