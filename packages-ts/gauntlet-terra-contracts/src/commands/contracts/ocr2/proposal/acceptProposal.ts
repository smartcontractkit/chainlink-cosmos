import { Result } from '@chainlink/gauntlet-core'
import { logger, prompt, diff } from '@chainlink/gauntlet-core/dist/utils'
import { TransactionResponse, RDD } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../../../lib/constants'
import {
  AbstractInstruction,
  instructionToCommand,
  BeforeExecute,
  Validation,
} from '../../../abstract/executionWrapper'
import { serializeOffchainConfig, deserializeConfig } from '../../../../lib/encoding'
import { getOffchainConfigInput, OffchainConfig, prepareOffchainConfigForDiff } from '../proposeOffchainConfig'
import { getLatestOCRConfigEvent } from '../../../../lib/inspection'
import assert from 'assert'

export type CommandInput = {
  proposalId: string
  digest: string
  offchainConfig: OffchainConfig
  randomSecret: string
}

type ContractInput = {
  id: string
  digest: string
}

const validations: Validation<CommandInput>[] = [
  {
    id: 'proposalId',
    msgSuccess: 'Config Proposal ID is provided',
    msgFail: 'Config Proposal ID is required',
    validate: () => async (input) => {
      if (!input.proposalId) return false
      return true
    },
  },
  {
    id: 'digest',
    msgSuccess: 'Config digest is provided',
    msgFail: 'Config digest is required',
    validate: () => async (input) => {
      if (!input.digest) return false
      return true
    },
  },
  {
    id: 'randomSecret',
    msgSuccess: 'Random secret is provided',
    msgFail: 'Secret generated at proposing offchain config is required',
    validate: () => async (input) => {
      if (!input.randomSecret) return false
      return true
    },
  },
  {
    id: 'OCRConfig',
    msgSuccess: 'Generated configuration matches with onchain proposal configuration',
    msgFail: 'Generated configuration does not correspond to the proposal configuration',
    validate: (context) => async (input) => {
      const { offchainConfig } = await serializeOffchainConfig(
        input.offchainConfig,
        process.env.SECRET!,
        input.randomSecret,
      )
      const proposal: any = await context.provider.wasm.contractQuery(context.contract, {
        proposal: {
          id: input.proposalId,
        },
      })

      try {
        assert.equal(offchainConfig.toString('base64'), proposal.offchain_config)
      } catch (err) {
        return false
      }

      return true
    },
  },
]

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  const { rdd: rddPath, secret } = flags

  if (!secret) {
    throw new Error('--secret flag is required.')
  }

  if (!process.env.SECRET) {
    throw new Error('SECRET is not set in env!')
  }

  const rdd = RDD.getRDD(rddPath)
  const contract = args[0]

  return {
    proposalId: flags.proposalId || flags.configProposal, // --configProposal alias requested by eng ops
    digest: flags.digest,
    offchainConfig: getOffchainConfigInput(rdd, contract),
    randomSecret: secret,
  }
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context, inputContext) => async () => {
  const { proposalId } = inputContext.input

  const proposal: any = await context.provider.wasm.contractQuery(context.contract, {
    proposal: {
      id: proposalId,
    },
  })

  const tryDeserialize = (config: string): OffchainConfig => {
    try {
      return deserializeConfig(Buffer.from(config, 'base64'))
    } catch (e) {
      return {} as OffchainConfig
    }
  }
  // Config in Proposal
  const offchainConfigInProposal = tryDeserialize(proposal.offchain_config)
  const configInProposal = prepareOffchainConfigForDiff(offchainConfigInProposal, { f: proposal.f })

  // Config in contract
  const event = await getLatestOCRConfigEvent(context.provider, context.contract)
  const offchainConfigInContract = event?.offchain_config
    ? tryDeserialize(event.offchain_config[0])
    : ({} as OffchainConfig)
  const configInContract = prepareOffchainConfigForDiff(offchainConfigInContract, { f: event?.f[0] })

  logger.info('Review the configuration difference from contract and proposal: green - added, red - deleted.')
  diff.printDiff(configInContract, configInProposal)
  await prompt('Continue?')
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  return {
    id: input.proposalId,
    digest: Buffer.from(input.digest, 'hex').toString('base64'),
  }
}

const validateInput = (input: CommandInput): boolean => {
  return true
}

const afterExecute = () => async (response: Result<TransactionResponse>) => {
  logger.success(`Config Proposal accepted on tx ${response.responses[0].tx.hash}`)
  const events = response.responses[0].tx.events
  if (!events) {
    logger.error('Could not retrieve events from tx')
    return
  }
  const digest = events[0]['wasm-set_config'].latest_config_digest[0]
  return {
    digest,
  }
}

const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  examples: [
    'yarn gauntlet ocr2:accept_proposal --network=<NETWORK> --configProposal=<PROPOSAL_ID> --digest=<DIGEST> <CONTRACT_ADDRESS>',
  ],
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
  validations,
}

export default instructionToCommand(instruction)
