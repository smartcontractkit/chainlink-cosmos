import { Result } from '@chainlink/gauntlet-core'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { TransactionResponse, providerUtils, RDD } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../../../lib/constants'
import { AbstractInstruction, instructionToCommand, BeforeExecute } from '../../../abstract/executionWrapper'
import { serializeOffchainConfig, deserializeConfig, generateSecretEncryptions } from '../../../../lib/encoding'
import { getOffchainConfigInput, OffchainConfig } from '../proposeOffchainConfig'
import { getLatestOCRConfig, printDiff } from '../../../../lib/inspection'
import Long from 'long'
import assert from 'assert'

type CommandInput = {
  proposalId: string
  digest: string
  offchainConfig: OffchainConfig
}

type ContractInput = {
  id: string
  digest: string
}

const translateConfig = (rawOffchainConfig: any, additionalConfig?: any): OffchainConfig => {
  const res = {
    ...rawOffchainConfig,
    ...(additionalConfig || {}),
    offchainPublicKeys: rawOffchainConfig.offchainPublicKeys?.map((key) => Buffer.from(key).toString('hex')),
  }

  const longsToNumber = (obj) => {
    for (const [key, value] of Object.entries(obj)) {
      if (Long.isLong(value)) {
        obj[key] = (value as Long).toNumber()
      } else if (typeof value === 'object') {
        longsToNumber(value)
      }
    }
  }

  longsToNumber(res)
  return res as OffchainConfig
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  const { rdd: rddPath, proposalId } = flags

  if (!rddPath) {
    throw new Error('No RDD flag provided!')
  }

  if (!proposalId) {
    throw new Error('No proposalId flag provided!')
  }

  const rdd = RDD.getRDD(rddPath)
  const contract = args[0]

  return {
    proposalId: flags.proposalId,
    digest: flags.digest,
    offchainConfig: getOffchainConfigInput(rdd, contract),
  }
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context) => async () => {
  const { proposalId, offchainConfig: offchainLocalConfig } = context.input

  const serializedLocalConfig = await serializeOffchainConfig(offchainLocalConfig)
  const localConfig = serializedLocalConfig.toString('base64')

  // Config in Proposal
  const proposal: any = await context.provider.wasm.contractQuery(context.contract, {
    proposal: {
      id: proposalId,
    },
  })
  const offchainConfigInProposal = await deserializeConfig(Buffer.from(proposal.offchain_config, 'base64'))
  const configInProposal = translateConfig(offchainConfigInProposal, { f: proposal.f })

  try {
    // assert.equal(localConfig, proposal.offchain_config)
  } catch (err) {
    logger.error("RDD configuration doesn't correspond the proposal configuration!")
    throw err
  }

  // Config in contract
  const event = await getLatestOCRConfig(context.provider, context.contract)
  const offchainConfigInContract = event?.offchain_config
    ? await deserializeConfig(Buffer.from(event.offchain_config[0], 'base64'))
    : ({} as OffchainConfig)
  const configInContract = translateConfig(offchainConfigInContract, { f: event?.f[0] })

  logger.info(
    'Review the configuration difference from contract and proposal: green - added, red - deleted.',
  )
  printDiff(configInContract, configInProposal)
  await prompt('Continue?')
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

const afterExecute = (response: Result<TransactionResponse>) => {
  const events = response.responses[0].tx.events
  if (!events) {
    logger.error('Could not retrieve events from tx')
    return
  }

  const digest = events[0]['wasm-set_config'].latest_config_digest[0]
  logger.success(`Proposal accepted`)
  logger.line()
  logger.info('Important: To inspect the aggregator, save the following DIGEST:')
  logger.info(digest)
  logger.line()
  return {
    digest,
  }
}

// yarn gauntlet ocr2:accept_proposal --network=bombay-testnet --id=4 --digest=71e6969c14c3e0cd47d75da229dbd2f76fd0f3c17e05635f78ac755a99897a2f terra14nrtuhrrhl2ldad7gln5uafgl8s2m25du98hlx
const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  instruction: {
    contract: 'ocr2',
    function: 'accept_proposal',
    category: CATEGORIES.OCR,
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  beforeExecute,
  afterExecute,
}

export default instructionToCommand(instruction)
