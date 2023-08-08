import { RDD } from '@chainlink/gauntlet-cosmos'
import { CATEGORIES } from '../../../lib/constants'
import { getLatestOCRConfigEvent } from '../../../lib/inspection'
import { AbstractInstruction, BeforeExecute, instructionToCommand } from '../../abstract/executionWrapper'
import { logger, diff } from '@chainlink/gauntlet-core/dist/utils'

type OnchainConfig = any
export type CommandInput = {
  f: number
  proposalId: string
  signers: string[]
  transmitters: string[]
  payees: string[]
  onchainConfig: OnchainConfig
}

export type ContractInput = {
  f: number
  id: string
  onchain_config: string
  signers: string[]
  transmitters: string[]
  payees: string[]
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput

  if (flags.rdd) {
    const { rdd: rddPath } = flags
    const rdd = RDD.getRDD(rddPath)
    const contract = args[0]
    const aggregator = rdd.contracts[contract]
    const aggregatorOperators: any[] = aggregator.oracles.map((o) => rdd.operators[o.operator])
    const signers = aggregatorOperators.map((o) => o.ocr2OnchainPublicKey[0])
    const transmitters = aggregatorOperators.map((o) => o.ocrNodeAddress[0])
    const payees = aggregatorOperators.map((o) => o.adminAddress)
    return {
      f: aggregator.config.f,
      proposalId: flags.proposalId || flags.configProposal || flags.id, // -configProposal alias requested by eng ops
      signers,
      transmitters,
      payees,
      onchainConfig: '',
    }
  }

  return {
    f: parseInt(flags.f),
    proposalId: flags.proposalId,
    signers: flags.signers,
    transmitters: flags.transmitters,
    payees: flags.payees,
    onchainConfig: '',
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  const signers = input.signers.map((s) => Buffer.from(s.replace('ocr2on_cosmos_', ''), 'hex').toString('base64'))

  return {
    f: Number(input.f),
    id: input.proposalId,
    onchain_config: input.onchainConfig,
    signers: signers,
    transmitters: input.transmitters,
    payees: input.payees,
  }
}

const validateInput = (input: CommandInput): boolean => {
  if (3 * input.f >= input.signers.length)
    throw new Error(`Signers length needs to be higher than 3 * f (${3 * input.f}). Currently ${input.signers.length}`)

  if (input.signers.length !== input.transmitters.length)
    throw new Error(`Signers and Trasmitters length are different`)

  if (input.transmitters.length !== input.payees.length) throw new Error(`Trasmitters and Payees length are different`)

  if (new Set(input.signers).size !== input.signers.length) throw new Error(`Signers array contains duplicates`)

  if (new Set(input.transmitters).size !== input.transmitters.length)
    throw new Error(`Transmitters array contains duplicates`)

  return true
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context, input) => async () => {
  const event = await getLatestOCRConfigEvent(context.provider, context.contract)
  logger.loading(`Executing ${context.id} from contract ${context.contract}`)

  const contractConfig = {
    f: event?.attributes?.find(({ key }) => key === 'f')?.value,
    transmitters: event?.attributes?.find(({ key }) => key === 'transmitters')?.value,
    signers: event?.attributes
      ?.filter(({ key }) => key === 'signers')
      ?.map((s) => Buffer.from(s.value, 'hex').toString('base64')),
    payees: event?.attributes?.find(({ key }) => key === 'payees')?.value,
  }

  const proposedConfig = {
    f: input.contract.f,
    transmitters: input.contract.transmitters,
    signers: input.contract.signers,
    payees: input.contract.payees,
  }

  logger.info('Review the proposed changes below: green - added, red - deleted.')
  diff.printDiff(contractConfig, proposedConfig)
}

export const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  examples: ['yarn gauntlet ocr2:propose_config --network=<NETWORK> --configProposal=<PROPOSAL_ID> <CONTRACT_ADDRESS>'],
  instruction: {
    contract: 'ocr2',
    function: 'propose_config',
    category: CATEGORIES.OCR,
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  beforeExecute,
}

export default instructionToCommand(instruction)
