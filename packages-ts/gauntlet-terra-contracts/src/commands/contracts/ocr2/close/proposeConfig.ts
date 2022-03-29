import { RDD } from '@chainlink/gauntlet-terra'
import { CATEGORIES } from '../../../../lib/constants'
import { AbstractInstruction, BeforeExecute, instructionToCommand } from '../../../abstract/executionWrapper'
import { logger, prompt } from '@chainlink/gauntlet-core/dist/utils'
import { ContractInput } from '../proposeConfig'
import { MnemonicKey } from '@terra-money/terra.js'

type CommandInput = {
  proposalId: number
}

const makeCommandInput = async (flags: any, args: string[]): Promise<CommandInput> => {
  if (flags.input) return flags.input as CommandInput
  return {
    proposalId: flags.configProposal,
  }
}

const makeContractInput = async (input: CommandInput): Promise<ContractInput> => {
  const hexToBase64 = (s) => Buffer.from(s, 'hex').toString('base64')
  const randomAcc = () => new MnemonicKey().publicKey?.address()!
  const makeEmptyOracle = (n: number) => ({
    signer: hexToBase64(new Array(64).fill(n.toString(16)).join('')),
    transmitter: randomAcc(),
    payee: randomAcc(),
  })
  // > f * 3 oracles
  const oracles = new Array(4).fill('').map((_, i) => makeEmptyOracle(i))

  return {
    f: Number(1),
    id: input.proposalId,
    onchain_config: '',
    signers: oracles.map((o) => o.signer),
    transmitters: oracles.map((o) => o.transmitter),
    payees: oracles.map((o) => o.payee),
  }
}

const validateInput = (input: CommandInput): boolean => {
  return true
}

const beforeExecute: BeforeExecute<CommandInput, ContractInput> = (context, inputContext) => async () => {
  const rddContract = RDD.getContractFromRDD(RDD.getRDD(context.flags.rdd), context.contract)
  logger.info(`IMPORTANT: You are proposing an EMPTY configuration on the following contract:
    - Contract: ${rddContract.address} ${rddContract.description ? '- ' + rddContract.description : ''}
  `)
  await prompt('Continue?')
}

const instruction: AbstractInstruction<CommandInput, ContractInput> = {
  examples: [
    'yarn gauntlet ocr2:propose_config:close --network=<NETWORK> --configProposal=<PROPOSAL_ID> <CONTRACT_ADDRESS>',
  ],
  instruction: {
    contract: 'ocr2',
    function: 'propose_config',
    subInstruction: 'close',
    category: CATEGORIES.OCR,
  },
  makeInput: makeCommandInput,
  validateInput: validateInput,
  makeContractInput: makeContractInput,
  beforeExecute,
}

export default instructionToCommand(instruction)
