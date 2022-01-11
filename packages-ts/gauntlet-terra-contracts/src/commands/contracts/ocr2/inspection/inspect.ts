import { getRDD } from '../../../../lib/rdd'
import { InspectInstruction, instructionToInspectCommand } from '../../../abstract/wrapper'
import { getOffchainConfigInput, OffchainConfig } from '../setConfig'

type Input = {
  description: string
  decimals: string | number
  minAnswer: string | number
  maxAnswer: string | number
  transmitters: string[]
  billingAccessController: string
  requesterAccessController: string
  link: string
  offchainConfig: OffchainConfig
}

type OnchainData = any

const makeInput = async (flags: any): Promise<Input> => {
  if (flags.input) return flags.input as Input
  const rdd = getRDD(flags.rdd)
  const info = rdd.contracts[flags.state]
  const aggregatorOperators: string[] = info.oracles.map((o) => o.operator)
  const transmitters = aggregatorOperators.map((operator) => rdd.operators[operator].ocrNodeAddress[0])
  const billingAccessController = flags.billingAccessController || process.env.BILLING_ACCESS_CONTROLLER
  const requesterAccessController = flags.requesterAccessController || process.env.REQUESTER_ACCESS_CONTROLLER
  const link = flags.link || process.env.LINK
  const offchainConfig = getOffchainConfigInput(rdd, flags.state)
  return {
    description: info.name,
    decimals: info.decimals,
    minAnswer: info.minSubmissionValue,
    maxAnswer: info.maxSubmissionValue,
    transmitters,
    billingAccessController,
    requesterAccessController,
    link,
    offchainConfig,
  }
}

const inspect = (input: Input, data: OnchainData): boolean => {
  console.log(data)
  console.log(input)
  return true
}

const instruction: InspectInstruction<Input, OnchainData> = {
  instruction: {
    contract: 'ocr2',
    function: '', // latest_config_details
  },
  makeInput,
  inspect,
}

export default instructionToInspectCommand(instruction)
