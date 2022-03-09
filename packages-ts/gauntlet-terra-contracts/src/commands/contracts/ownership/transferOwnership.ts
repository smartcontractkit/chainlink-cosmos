import { AccAddress } from '@terra-money/terra.js'
import { RDD } from '@chainlink/gauntlet-terra'
import { AbstractInstruction, BeforeExecute } from '../../abstract/executionWrapper'
import { CATEGORIES } from '../../../lib/constants'
import { CONTRACT_LIST } from '../../../lib/contracts'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'

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

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context) => async () => {
  const currentOwner = await context.provider.wasm.contractQuery(context.contract, 'owner' as any)
  const contract = RDD.getContractFromRDD(RDD.getRDD(context.flags.rdd), context.contract)
  logger.info(`Proposing Ownership Transfer of contract of type "${contract.type}":
    - Contract: ${contract.address} ${contract.description ? '- ' + contract.description : ''}
    - Current Owner: ${currentOwner}
    - Next Owner: ${context.contractInput.to}
  `)
  await prompt('Continue?')
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
